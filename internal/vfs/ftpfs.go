package vfs

import (
	"bytes"
	"crypto/tls"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"time"

	"github.com/feherkaroly/vc/internal/config"
	"github.com/jlaffaye/ftp"
)

// FTPFS implements FileSystem over an FTP/FTPS connection.
type FTPFS struct {
	conn   *ftp.ServerConn
	mu     sync.Mutex
	done   chan struct{}
}

// NewFTPFS establishes an FTP/FTPS connection based on the given server config.
func NewFTPFS(cfg config.ServerConfig) (*FTPFS, error) {
	port := cfg.Port
	if port == 0 {
		port = 21
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, port)

	var opts []ftp.DialOption
	opts = append(opts, ftp.DialWithTimeout(10*time.Second))

	if cfg.Protocol == "ftps" {
		opts = append(opts, ftp.DialWithExplicitTLS(&tls.Config{
			InsecureSkipVerify: true,
		}))
	}

	conn, err := ftp.Dial(addr, opts...)
	if err != nil {
		return nil, fmt.Errorf("FTP dial %s: %w", addr, err)
	}

	if err := conn.Login(cfg.User, cfg.Password); err != nil {
		conn.Quit()
		return nil, fmt.Errorf("FTP login: %w", err)
	}

	f := &FTPFS{
		conn: conn,
		done: make(chan struct{}),
	}

	// Start NOOP keep-alive goroutine
	go f.keepAlive()

	return f, nil
}

func (f *FTPFS) keepAlive() {
	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ticker.C:
			f.mu.Lock()
			f.conn.NoOp()
			f.mu.Unlock()
		case <-f.done:
			return
		}
	}
}

func (f *FTPFS) ReadDir(dirPath string) ([]DirEntry, error) {
	f.mu.Lock()
	ftpEntries, err := f.conn.List(dirPath)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}

	entries := make([]DirEntry, 0, len(ftpEntries))
	for _, e := range ftpEntries {
		if e.Name == "." || e.Name == ".." {
			continue
		}
		entry := DirEntry{
			Name:    e.Name,
			Size:    int64(e.Size),
			ModTime: e.Time,
			Mode:    ftpEntryMode(e),
			IsDir:   e.Type == ftp.EntryTypeFolder,
		}
		if e.Type == ftp.EntryTypeLink {
			entry.IsLink = true
			entry.LinkTo = e.Target
		}
		entries = append(entries, entry)
	}
	return entries, nil
}

func (f *FTPFS) Stat(filePath string) (FileInfo, error) {
	f.mu.Lock()
	entry, err := f.conn.GetEntry(filePath)
	f.mu.Unlock()
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:    path.Base(filePath),
		Size:    int64(entry.Size),
		ModTime: entry.Time,
		Mode:    ftpEntryMode(entry),
		IsDir:   entry.Type == ftp.EntryTypeFolder,
	}, nil
}

func (f *FTPFS) Lstat(filePath string) (FileInfo, error) {
	return f.Stat(filePath)
}

func (f *FTPFS) Readlink(_ string) (string, error) {
	return "", fmt.Errorf("symlinks not supported over FTP")
}

func (f *FTPFS) Open(filePath string) (io.ReadCloser, error) {
	f.mu.Lock()
	resp, err := f.conn.Retr(filePath)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	return resp, nil
}

// ftpWriteCloser wraps an io.PipeWriter and waits for the background Stor goroutine to finish.
type ftpWriteCloser struct {
	pw   *io.PipeWriter
	done chan error
}

func (w *ftpWriteCloser) Write(p []byte) (int, error) {
	return w.pw.Write(p)
}

func (w *ftpWriteCloser) Close() error {
	w.pw.Close()
	return <-w.done
}

func (f *FTPFS) Create(filePath string, _ fs.FileMode) (io.WriteCloser, error) {
	pr, pw := io.Pipe()
	done := make(chan error, 1)

	go func() {
		f.mu.Lock()
		err := f.conn.Stor(filePath, pr)
		f.mu.Unlock()
		pr.CloseWithError(err)
		done <- err
	}()

	return &ftpWriteCloser{pw: pw, done: done}, nil
}

func (f *FTPFS) MkdirAll(dirPath string, _ fs.FileMode) error {
	// Walk path components and create each if needed
	parts := strings.Split(strings.Trim(dirPath, "/"), "/")
	current := "/"
	for _, part := range parts {
		if part == "" {
			continue
		}
		current = path.Join(current, part)
		f.mu.Lock()
		err := f.conn.MakeDir(current)
		f.mu.Unlock()
		if err != nil {
			// Ignore "already exists" errors
			if !strings.Contains(err.Error(), "exists") &&
				!strings.Contains(err.Error(), "550") {
				return err
			}
		}
	}
	return nil
}

func (f *FTPFS) Remove(filePath string) error {
	f.mu.Lock()
	err := f.conn.Delete(filePath)
	f.mu.Unlock()
	return err
}

func (f *FTPFS) RemoveAll(filePath string) error {
	fi, err := f.Stat(filePath)
	if err != nil {
		return nil // Already gone
	}

	if !fi.IsDir {
		f.mu.Lock()
		err := f.conn.Delete(filePath)
		f.mu.Unlock()
		return err
	}

	entries, err := f.ReadDir(filePath)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		childPath := path.Join(filePath, entry.Name)
		if err := f.RemoveAll(childPath); err != nil {
			return err
		}
	}

	f.mu.Lock()
	err = f.conn.RemoveDirRecur(filePath)
	f.mu.Unlock()
	if err != nil {
		// Fallback: try simple RemoveDir
		f.mu.Lock()
		err = f.conn.RemoveDir(filePath)
		f.mu.Unlock()
	}
	return err
}

func (f *FTPFS) Rename(oldpath, newpath string) error {
	f.mu.Lock()
	err := f.conn.Rename(oldpath, newpath)
	f.mu.Unlock()
	return err
}

func (f *FTPFS) ReadFile(filePath string) ([]byte, error) {
	f.mu.Lock()
	resp, err := f.conn.Retr(filePath)
	f.mu.Unlock()
	if err != nil {
		return nil, err
	}
	defer resp.Close()

	var buf bytes.Buffer
	if _, err := io.Copy(&buf, resp); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (f *FTPFS) Walk(root string, fn WalkFunc) error {
	return f.walkDir(root, fn)
}

func (f *FTPFS) walkDir(dir string, fn WalkFunc) error {
	fi, err := f.Stat(dir)
	if err != nil {
		return fn(dir, FileInfo{}, err)
	}

	if err := fn(dir, fi, nil); err != nil {
		return err
	}

	if !fi.IsDir {
		return nil
	}

	entries, err := f.ReadDir(dir)
	if err != nil {
		return fn(dir, fi, err)
	}

	for _, entry := range entries {
		childPath := path.Join(dir, entry.Name)
		if entry.IsDir {
			if err := f.walkDir(childPath, fn); err != nil {
				return err
			}
		} else {
			childInfo := FileInfo{
				Name:    entry.Name,
				Size:    entry.Size,
				ModTime: entry.ModTime,
				Mode:    entry.Mode,
				IsDir:   false,
			}
			if err := fn(childPath, childInfo, nil); err != nil {
				return err
			}
		}
	}
	return nil
}

func (f *FTPFS) Join(elem ...string) string {
	return path.Join(elem...)
}

func (f *FTPFS) Dir(p string) string {
	return path.Dir(p)
}

func (f *FTPFS) Base(p string) string {
	return path.Base(p)
}

func (f *FTPFS) IsLocal() bool {
	return false
}

func (f *FTPFS) Close() error {
	close(f.done)
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.conn.Quit()
}

// ftpEntryMode converts an FTP entry to an approximate os.FileMode.
func ftpEntryMode(e *ftp.Entry) os.FileMode {
	var mode os.FileMode = 0644
	if e.Type == ftp.EntryTypeFolder {
		mode = os.ModeDir | 0755
	} else if e.Type == ftp.EntryTypeLink {
		mode = os.ModeSymlink | 0777
	}
	return mode
}
