package fileops

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Copy recursively copies src to dst.
func Copy(ctx context.Context, src, dst string, onProgress func(Progress)) error {
	srcInfo, err := os.Lstat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}

	if srcInfo.IsDir() {
		return copyDir(ctx, src, dst, onProgress)
	}
	return copyFile(ctx, src, dst, srcInfo, onProgress)
}

func copyFile(ctx context.Context, src, dst string, srcInfo os.FileInfo, onProgress func(Progress)) error {
	// Ensure destination directory exists
	if err := os.MkdirAll(filepath.Dir(dst), 0755); err != nil {
		return err
	}

	sf, err := os.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	df, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, srcInfo.Mode())
	if err != nil {
		return err
	}
	defer df.Close()

	total := srcInfo.Size()
	var copied int64

	buf := make([]byte, 32*1024)
	for {
		// Check for cancellation
		if err := ctx.Err(); err != nil {
			df.Close()
			os.Remove(dst)
			return err
		}

		n, readErr := sf.Read(buf)
		if n > 0 {
			if _, writeErr := df.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			copied += int64(n)
			if onProgress != nil {
				onProgress(Progress{
					FileName: filepath.Base(src),
					Total:    total,
					Done:     copied,
				})
			}
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			return readErr
		}
	}

	return nil
}

func copyDir(ctx context.Context, src, dst string, onProgress func(Progress)) error {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dst, srcInfo.Mode()); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())

		if err := Copy(ctx, srcPath, dstPath, onProgress); err != nil {
			return err
		}
	}

	return nil
}
