package vfs

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
)

// LocalFS implements FileSystem for the local OS filesystem.
type LocalFS struct{}

// NewLocalFS returns a new LocalFS instance.
func NewLocalFS() *LocalFS {
	return &LocalFS{}
}

func (l *LocalFS) ReadDir(path string) ([]DirEntry, error) {
	osEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}

	entries := make([]DirEntry, 0, len(osEntries))
	for _, de := range osEntries {
		info, err := de.Info()
		if err != nil {
			continue
		}

		entry := DirEntry{
			Name:    de.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			IsDir:   de.IsDir(),
		}

		if de.Type()&os.ModeSymlink != 0 {
			entry.IsLink = true
			if target, err := os.Readlink(filepath.Join(path, de.Name())); err == nil {
				entry.LinkTo = target
			}
			if fi, err := os.Stat(filepath.Join(path, de.Name())); err == nil {
				entry.IsDir = fi.IsDir()
				entry.Size = fi.Size()
				entry.ModTime = fi.ModTime()
				entry.Mode = fi.Mode()
			}
		}

		entries = append(entries, entry)
	}
	return entries, nil
}

func (l *LocalFS) Stat(path string) (FileInfo, error) {
	fi, err := os.Stat(path)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:    fi.Name(),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
		Mode:    fi.Mode(),
		IsDir:   fi.IsDir(),
	}, nil
}

func (l *LocalFS) Lstat(path string) (FileInfo, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return FileInfo{}, err
	}
	return FileInfo{
		Name:    fi.Name(),
		Size:    fi.Size(),
		ModTime: fi.ModTime(),
		Mode:    fi.Mode(),
		IsDir:   fi.IsDir(),
	}, nil
}

func (l *LocalFS) Readlink(path string) (string, error) {
	return os.Readlink(path)
}

func (l *LocalFS) Open(path string) (io.ReadCloser, error) {
	return os.Open(path)
}

func (l *LocalFS) Create(path string, mode fs.FileMode) (io.WriteCloser, error) {
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
}

func (l *LocalFS) MkdirAll(path string, perm fs.FileMode) error {
	return os.MkdirAll(path, perm)
}

func (l *LocalFS) Remove(path string) error {
	return os.Remove(path)
}

func (l *LocalFS) RemoveAll(path string) error {
	return os.RemoveAll(path)
}

func (l *LocalFS) Rename(oldpath, newpath string) error {
	return os.Rename(oldpath, newpath)
}

func (l *LocalFS) ReadFile(path string) ([]byte, error) {
	return os.ReadFile(path)
}

func (l *LocalFS) Walk(root string, fn WalkFunc) error {
	return filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return fn(path, FileInfo{}, err)
		}
		return fn(path, FileInfo{
			Name:    info.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			IsDir:   info.IsDir(),
		}, nil)
	})
}

func (l *LocalFS) Join(elem ...string) string {
	return filepath.Join(elem...)
}

func (l *LocalFS) Dir(path string) string {
	return filepath.Dir(path)
}

func (l *LocalFS) Base(path string) string {
	return filepath.Base(path)
}

func (l *LocalFS) IsLocal() bool {
	return true
}

func (l *LocalFS) Close() error {
	return nil
}
