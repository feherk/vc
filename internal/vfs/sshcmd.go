package vfs

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/feherkaroly/vc/internal/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// ExpandKeyPath resolves "~/..." to the home directory and a bare file name
// (e.g. "id_yk") to ~/.ssh/<name>.
func ExpandKeyPath(p string) string {
	p = strings.TrimSpace(p)
	if p == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return p
	}
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") || strings.HasPrefix(p, `~\`) {
		return filepath.Join(home, p[2:])
	}
	if !strings.ContainsAny(p, `/\`) {
		return filepath.Join(home, ".ssh", p)
	}
	return p
}

// UsesSystemSSH reports whether the SFTP connection must go through the
// system ssh binary instead of the built-in client: when explicitly requested,
// when no credentials are given (so ~/.ssh/config and the agent decide), or
// when the key cannot be used directly — hardware security keys (YubiKey,
// sk-ssh-ed25519) and passphrase-protected keys.
func UsesSystemSSH(cfg config.ServerConfig) bool {
	if cfg.Protocol != "sftp" {
		return false
	}
	if cfg.SystemSSH || cfg.Via != "" {
		return true
	}
	if cfg.KeyPath == "" {
		return cfg.Password == ""
	}
	keyData, err := os.ReadFile(ExpandKeyPath(cfg.KeyPath))
	if err != nil {
		return false // the built-in client reports the read error
	}
	_, err = ssh.ParsePrivateKey(keyData)
	return err != nil
}

// findSSH prefers Homebrew's OpenSSH on macOS: Apple's build has no
// FIDO/security key support.
func findSSH() (string, error) {
	for _, p := range []string{"/opt/homebrew/bin/ssh", "/usr/local/bin/ssh"} {
		if fi, err := os.Stat(p); err == nil && !fi.IsDir() {
			return p, nil
		}
	}
	p, err := exec.LookPath("ssh")
	if err != nil {
		return "", fmt.Errorf("ssh binary not found in PATH")
	}
	return p, nil
}

// stderrTap forwards ssh's stderr to the terminal while connecting (PIN and
// touch prompts), then only keeps the tail for error messages.
type stderrTap struct {
	mu   sync.Mutex
	out  io.Writer
	tail []byte
}

func (t *stderrTap) Write(p []byte) (int, error) {
	t.mu.Lock()
	defer t.mu.Unlock()
	if t.out != nil {
		t.out.Write(p)
	}
	t.tail = append(t.tail, p...)
	if len(t.tail) > 4096 {
		t.tail = t.tail[len(t.tail)-4096:]
	}
	return len(p), nil
}

func (t *stderrTap) detach() {
	t.mu.Lock()
	t.out = nil
	t.mu.Unlock()
}

func (t *stderrTap) lastLine() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	lines := strings.Split(strings.TrimSpace(string(t.tail)), "\n")
	return strings.TrimSpace(lines[len(lines)-1])
}

// shellQuote quotes s for a POSIX shell (the remote login shell on the Via
// host parses the relayed command line).
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'\''`) + "'"
}

// sshArgs builds the system ssh command line. Without Via:
// `ssh [-p port] [-l user] [-i key] -s -- host sftp`. With Via the local ssh
// logs in to the Via host (Key Path applies to this hop) and runs the
// `ssh -s … sftp` to Host there, non-interactively, with that machine's
// own keys and ~/.ssh/config.
func sshArgs(cfg config.ServerConfig) []string {
	target := []string{}
	if cfg.Port != 0 {
		target = append(target, "-p", strconv.Itoa(cfg.Port))
	}
	if cfg.User != "" {
		target = append(target, "-l", cfg.User)
	}
	args := []string{"-o", "ConnectTimeout=10", "-o", "ServerAliveInterval=30"}
	if kp := ExpandKeyPath(cfg.KeyPath); kp != "" {
		args = append(args, "-i", kp, "-o", "IdentitiesOnly=yes")
	}
	if cfg.Via == "" {
		args = append(args, target...)
		return append(args, "-s", "--", cfg.Host, "sftp")
	}
	remote := []string{"ssh", "-o", "BatchMode=yes", "-o", "ConnectTimeout=10", "-o", "ServerAliveInterval=30"}
	remote = append(remote, target...)
	remote = append(remote, "-s", "--", cfg.Host, "sftp")
	quoted := make([]string, len(remote))
	for i, r := range remote {
		quoted[i] = shellQuote(r)
	}
	return append(args, "--", cfg.Via, strings.Join(quoted, " "))
}

// newSystemSSHSFTP runs `ssh -s host sftp` (or, with Via, the same on the
// Via host) and speaks SFTP over its pipes. ssh reads PIN / passphrase from
// the controlling terminal, so the caller must suspend the TUI while this
// runs.
func newSystemSSHSFTP(cfg config.ServerConfig) (*SFTPFS, error) {
	sshBin, err := findSSH()
	if err != nil {
		return nil, err
	}

	cmd := exec.Command(sshBin, sshArgs(cfg)...)
	tap := &stderrTap{out: os.Stderr}
	cmd.Stderr = tap
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start ssh: %w", err)
	}

	client, err := sftp.NewClientPipe(stdout, stdin, sftp.UseConcurrentWrites(true))
	if err != nil {
		stdin.Close()
		cmd.Process.Kill()
		cmd.Wait()
		if msg := tap.lastLine(); msg != "" {
			return nil, fmt.Errorf("ssh: %s", msg)
		}
		return nil, fmt.Errorf("SFTP over ssh: %w", err)
	}
	tap.detach()

	return &SFTPFS{client: client, cmd: cmd}, nil
}

// stopCmd waits briefly for ssh to exit after its stdin closed, then kills it.
func stopCmd(cmd *exec.Cmd) {
	done := make(chan struct{})
	go func() {
		cmd.Wait()
		close(done)
	}()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		cmd.Process.Kill()
		<-done
	}
}
