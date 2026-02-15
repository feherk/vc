package fileops

import (
	"os"
	"path/filepath"
)

// CalcDirSize recursively calculates the total size of a directory.
func CalcDirSize(path string) int64 {
	var total int64
	filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir() {
			total += info.Size()
		}
		return nil
	})
	return total
}
