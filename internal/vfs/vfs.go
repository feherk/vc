package vfs

import (
	"io"
	"io/fs"
	"os"
	"time"
)

// FileInfo holds metadata about a file or directory.
type FileInfo struct {
	Name    string
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
	IsDir   bool
}

// DirEntry represents a single entry when listing a directory.
type DirEntry struct {
	Name    string
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
	IsDir   bool
	IsLink  bool
	LinkTo  string
}

// WalkFunc is the callback for Walk.
type WalkFunc func(path string, info FileInfo, err error) error

// FileSystem is the abstraction layer for local and remote filesystems.
type FileSystem interface {
	ReadDir(path string) ([]DirEntry, error)
	Stat(path string) (FileInfo, error)
	Lstat(path string) (FileInfo, error)
	Readlink(path string) (string, error)
	Open(path string) (io.ReadCloser, error)
	Create(path string, mode fs.FileMode) (io.WriteCloser, error)
	MkdirAll(path string, perm fs.FileMode) error
	Remove(path string) error
	RemoveAll(path string) error
	Rename(oldpath, newpath string) error
	ReadFile(path string) ([]byte, error)
	Walk(root string, fn WalkFunc) error
	Join(elem ...string) string
	Dir(path string) string
	Base(path string) string
	IsLocal() bool
	Close() error
}
