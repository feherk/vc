package vfs

import (
	"strings"
	"testing"

	"github.com/feherkaroly/vc/internal/config"
)

func TestSSHArgsDirect(t *testing.T) {
	got := strings.Join(sshArgs(config.ServerConfig{Protocol: "sftp", Host: "h", Port: 2222, User: "u"}), " ")
	want := "-o ConnectTimeout=10 -o ServerAliveInterval=30 -p 2222 -l u -s -- h sftp"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
}

func TestSSHArgsVia(t *testing.T) {
	args := sshArgs(config.ServerConfig{Protocol: "sftp", Host: "h", User: "u", Via: "relay", KeyPath: "/k"})
	got := strings.Join(args, " ")
	want := "-o ConnectTimeout=10 -o ServerAliveInterval=30 -i /k -o IdentitiesOnly=yes -- relay " +
		"'ssh' '-o' 'BatchMode=yes' '-o' 'ConnectTimeout=10' '-o' 'ServerAliveInterval=30' '-l' 'u' '-s' '--' 'h' 'sftp'"
	if got != want {
		t.Fatalf("got %q\nwant %q", got, want)
	}
	if !UsesSystemSSH(config.ServerConfig{Protocol: "sftp", Host: "h", Password: "p", Via: "relay"}) {
		t.Fatal("Via must force the system ssh")
	}
}

func TestShellQuote(t *testing.T) {
	if q := shellQuote("it's"); q != `'it'\''s'` {
		t.Fatalf("got %q", q)
	}
}
