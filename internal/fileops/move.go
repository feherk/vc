package fileops

import (
	"context"

	"github.com/feherkaroly/vc/internal/vfs"
)

// Move moves or renames src to dst.
// If srcFS and dstFS are the same local filesystem, tries Rename first (fast path).
// Falls back to copy+delete for cross-filesystem moves.
func Move(ctx context.Context, srcFS vfs.FileSystem, src string, dstFS vfs.FileSystem, dst string, onProgress func(Progress)) error {
	// Ensure destination directory exists
	if err := dstFS.MkdirAll(dstFS.Dir(dst), 0755); err != nil {
		return err
	}

	// Fast path: same filesystem → rename
	if srcFS == dstFS {
		err := srcFS.Rename(src, dst)
		if err == nil {
			return nil
		}
	}

	// Fallback: copy then delete
	if err := Copy(ctx, srcFS, src, dstFS, dst, onProgress); err != nil {
		// On cancel, clean up partial copy but keep source
		if ctx.Err() != nil {
			dstFS.RemoveAll(dst)
		}
		return err
	}
	return srcFS.RemoveAll(src)
}
