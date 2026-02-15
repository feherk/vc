package fileops

import "github.com/feherkaroly/vc/internal/vfs"

// CalcDirSize recursively calculates the total size of a directory.
func CalcDirSize(fs vfs.FileSystem, path string) int64 {
	var total int64
	fs.Walk(path, func(_ string, info vfs.FileInfo, err error) error {
		if err != nil {
			return nil // Skip errors
		}
		if !info.IsDir {
			total += info.Size
		}
		return nil
	})
	return total
}
