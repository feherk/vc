package fileops

import "os"

// Delete removes a file or directory recursively.
func Delete(path string) error {
	return os.RemoveAll(path)
}
