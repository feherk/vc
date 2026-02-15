package fileops

import (
	"context"
	"os"
	"path/filepath"
)

// Move moves or renames src to dst.
// First tries os.Rename (fast, same filesystem).
// Falls back to copy+delete for cross-filesystem moves.
func Move(ctx context.Context, src, dst string, onProgress func(Progress)) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	err := os.Rename(src, dst)
	if err == nil {
		return nil
	}

	// Fallback: copy then delete
	if err := Copy(ctx, src, dst, onProgress); err != nil {
		// On cancel, clean up partial copy but keep source
		if ctx.Err() != nil {
			os.RemoveAll(dst)
		}
		return err
	}
	return os.RemoveAll(src)
}
