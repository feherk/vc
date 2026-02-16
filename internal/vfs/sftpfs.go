package vfs

import (
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"path"
	"strings"

	"github.com/feherkaroly/vc/internal/config"
	"github.com/pkg/sftp"
	"golang.org/x/crypto/ssh"
)

// SFTPFS implements FileSystem over an SSH/SFTP connection.
type SFTPFS struct {
	client    *sftp.Client
	sshClient *ssh.Client
}

// NewSFTPFS establishes an SFTP connection based on the given server config.
func NewSFTPFS(cfg config.ServerConfig) (*SFTPFS, error) {
	port := cfg.Port
	if port == 0 {
		port = 22
	}

	authMethods := []ssh.AuthMethod{}

	// Private key authentication
	if cfg.KeyPath != "" {
		keyData, err := os.ReadFile(cfg.KeyPath)
		if err != nil {
			return nil, fmt.Errorf("read key %s: %w", cfg.KeyPath, err)
		}
		var signer ssh.Signer
		signer, err = ssh.ParsePrivateKey(keyData)
		if err != nil {
			return nil, fmt.Errorf("parse key: %w", err)
		}
		authMethods = append(authMethods, ssh.PublicKeys(signer))
	}

	// Password authentication
	if cfg.Password != "" {
		authMethods = append(authMethods, ssh.Password(cfg.Password))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method configured")
	}

	sshConfig := &ssh.ClientConfig{
		User:            cfg.User,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, port)
	sshClient, err := ssh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("SSH dial %s: %w", addr, err)
	}

	sftpClient, err := sftp.NewClient(sshClient, sftp.UseConcurrentWrites(true))
	if err != nil {
		sshClient.Close()
		return nil, fmt.Errorf("SFTP client: %w", err)
	}

	return &SFTPFS{
		client:    sftpClient,
		sshClient: sshClient,
	}, nil
}

func (s *SFTPFS) ReadDir(dirPath string) ([]DirEntry, error) {
	infos, err := s.client.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	entries := make([]DirEntry, 0, len(infos))
	for _, fi := range infos {
		entry := DirEntry{
			Name:    fi.Name(),
			Size:    fi.Size(),
			ModTime: fi.ModTime(),
			Mode:    fi.Mode(),
			IsDir:   fi.IsDir(),
		}

		if fi.Mode()&os.ModeSymlink != 0 {
			entry.IsLink = true
			if target, err := s.client.ReadLink(path.Join(dirPath, fi.Name())); err == nil {
				entry.LinkTo = target
			}
			if resolved, err := s.client.Stat(path.Join(dirPath, fi.Name())); err == nil {
				entry.IsDir = resolved.IsDir()
				entry.Size = resolved.Size()
				entry.ModTime = resolved.ModTime()
				entry.Mode = resolved.Mode()
			}
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

func (s *SFTPFS) Stat(filePath string) (FileInfo, error) {
	fi, err := s.client.Stat(filePath)
	if err != nil {
		return FileInfo{}, err
	}
	return fileInfoFromOS(fi), nil
}

func (s *SFTPFS) Lstat(filePath string) (FileInfo, error) {
	fi, err := s.client.Lstat(filePath)
	if err != nil {
		return FileInfo{}, err
	}
	return fileInfoFromOS(fi), nil
}

func (s *SFTPFS) Readlink(filePath string) (string, error) {
	return s.client.ReadLink(filePath)
}

func (s *SFTPFS) Open(filePath string) (io.ReadCloser, error) {
	return s.client.Open(filePath)
}

func (s *SFTPFS) Create(filePath string, mode fs.FileMode) (io.WriteCloser, error) {
	f, err := s.client.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC)
	if err != nil {
		return nil, err
	}
	if err := s.client.Chmod(filePath, mode); err != nil {
		// Best effort — some servers may not support chmod
		_ = err
	}
	return f, nil
}

func (s *SFTPFS) MkdirAll(dirPath string, perm fs.FileMode) error {
	return s.client.MkdirAll(dirPath)
}

func (s *SFTPFS) Remove(filePath string) error {
	return s.client.Remove(filePath)
}

// RemoveAll recursively removes a file or directory.
func (s *SFTPFS) RemoveAll(filePath string) error {
	fi, err := s.client.Lstat(filePath)
	if err != nil {
		if isNotExist(err) {
			return nil
		}
		return err
	}

	if !fi.IsDir() {
		return s.client.Remove(filePath)
	}

	entries, err := s.client.ReadDir(filePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childPath := path.Join(filePath, entry.Name())
		if err := s.RemoveAll(childPath); err != nil {
			return err
		}
	}

	return s.client.RemoveDirectory(filePath)
}

func (s *SFTPFS) Rename(oldpath, newpath string) error {
	return s.client.Rename(oldpath, newpath)
}

func (s *SFTPFS) ReadFile(filePath string) ([]byte, error) {
	f, err := s.client.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return io.ReadAll(f)
}

func (s *SFTPFS) Walk(root string, fn WalkFunc) error {
	return s.walk(root, fn)
}

func (s *SFTPFS) walk(dir string, fn WalkFunc) error {
	fi, err := s.client.Stat(dir)
	if err != nil {
		return fn(dir, FileInfo{}, err)
	}

	info := fileInfoFromOS(fi)
	err = fn(dir, info, nil)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return nil
	}

	entries, err := s.client.ReadDir(dir)
	if err != nil {
		return fn(dir, info, err)
	}

	for _, entry := range entries {
		childPath := path.Join(dir, entry.Name())
		if entry.IsDir() {
			if err := s.walk(childPath, fn); err != nil {
				return err
			}
		} else {
			childInfo := fileInfoFromOS(entry)
			if err := fn(childPath, childInfo, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *SFTPFS) Join(elem ...string) string {
	return path.Join(elem...)
}

func (s *SFTPFS) Dir(p string) string {
	return path.Dir(p)
}

func (s *SFTPFS) Base(p string) string {
	return path.Base(p)
}

func (s *SFTPFS) IsLocal() bool {
	return false
}

func (s *SFTPFS) Close() error {
	s.client.Close()
	return s.sshClient.Close()
}

func fileInfoFromOS(fi os.FileInfo) FileInfo {
	return FileInfo{
		Name:    fi.Name(),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
		Mode:    fi.Mode(),
		IsDir:   fi.IsDir(),
	}
}

func isNotExist(err error) bool {
	if os.IsNotExist(err) {
		return true
	}
	// sftp may return "file does not exist" as a string
	if err != nil && strings.Contains(err.Error(), "not exist") {
		return true
	}
	// Check for network errors
	if _, ok := err.(*net.OpError); ok {
		return false
	}
	return false
}
