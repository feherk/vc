package fileops

import "os"

// MkDir creates a new directory (with parents if needed).
func MkDir(path string) error {
	return os.MkdirAll(path, 0755)
}
