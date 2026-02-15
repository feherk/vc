package fileops

import "github.com/feherkaroly/vc/internal/vfs"

// Delete removes a file or directory recursively.
func Delete(fs vfs.FileSystem, path string) error {
	return fs.RemoveAll(path)
}
