package fileops

import "github.com/feherkaroly/vc/internal/vfs"

// MkDir creates a new directory (with parents if needed).
func MkDir(fs vfs.FileSystem, path string) error {
	return fs.MkdirAll(path, 0755)
}
