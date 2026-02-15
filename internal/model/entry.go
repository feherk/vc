package model

import (
	"os"
	"time"
)

// FileEntry represents a single file or directory entry in a panel.
type FileEntry struct {
	Name    string
	Size    int64
	ModTime time.Time
	Mode    os.FileMode
	IsDir   bool
	IsLink  bool
	LinkTo  string

	// Calculated directory size (via Space key), -1 means not calculated
	DirSize int64
}

// DisplaySize returns the size to show. For directories, it returns
// DirSize if calculated, otherwise -1.
func (e *FileEntry) DisplaySize() int64 {
	if e.IsDir {
		return e.DirSize
	}
	return e.Size
}
