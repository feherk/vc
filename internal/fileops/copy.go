package fileops

import (
	"context"
	"fmt"
	"io"

	"github.com/feherkaroly/vc/internal/vfs"
)

// Copy recursively copies src to dst, supporting cross-filesystem operations.
// When preserveMode is true, the source file/directory permissions are preserved.
// When false, default permissions are used (0666 for files, 0777 for dirs, modified by umask).
func Copy(ctx context.Context, srcFS vfs.FileSystem, src string, dstFS vfs.FileSystem, dst string, preserveMode bool, onProgress func(Progress)) error {
	srcInfo, err := srcFS.Lstat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %w", src, err)
	}

	if srcInfo.IsDir {
		return copyDir(ctx, srcFS, src, dstFS, dst, preserveMode, onProgress)
	}
	return copyFile(ctx, srcFS, src, dstFS, dst, srcInfo, preserveMode, onProgress)
}

func copyFile(ctx context.Context, srcFS vfs.FileSystem, src string, dstFS vfs.FileSystem, dst string, srcInfo vfs.FileInfo, preserveMode bool, onProgress func(Progress)) error {
	// Ensure destination directory exists
	if err := dstFS.MkdirAll(dstFS.Dir(dst), 0755); err != nil {
		return err
	}

	sf, err := srcFS.Open(src)
	if err != nil {
		return err
	}
	defer sf.Close()

	mode := srcInfo.Mode
	if !preserveMode {
		mode = 0666
	}
	df, err := dstFS.Create(dst, mode)
	if err != nil {
		return err
	}
	defer df.Close()

	// Fast path: use io.Copy which leverages sftp.File's concurrent ReadFrom/WriteTo
	if onProgress == nil {
		_, err := io.Copy(df, sf)
		return err
	}

	total := srcInfo.Size
	var copied int64

	buf := make([]byte, 256*1024)
	for {
		if err := ctx.Err(); err != nil {
			df.Close()
			dstFS.Remove(dst)
			return err
		}

		n, readErr := sf.Read(buf)
		if n > 0 {
			if _, writeErr := df.Write(buf[:n]); writeErr != nil {
				return writeErr
			}
			copied += int64(n)
			onProgress(Progress{
				FileName: srcFS.Base(src),
				Total:    total,
				Done:     copied,
			})
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

func copyDir(ctx context.Context, srcFS vfs.FileSystem, src string, dstFS vfs.FileSystem, dst string, preserveMode bool, onProgress func(Progress)) error {
	srcInfo, err := srcFS.Stat(src)
	if err != nil {
		return err
	}

	dirMode := srcInfo.Mode
	if !preserveMode {
		dirMode = 0777
	}
	if err := dstFS.MkdirAll(dst, dirMode); err != nil {
		return err
	}

	entries, err := srcFS.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if err := ctx.Err(); err != nil {
			return err
		}

		srcPath := srcFS.Join(src, entry.Name)
		dstPath := dstFS.Join(dst, entry.Name)

		if err := Copy(ctx, srcFS, srcPath, dstFS, dstPath, preserveMode, onProgress); err != nil {
			return err
		}
	}

	return nil
}
