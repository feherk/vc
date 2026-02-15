//go:build !windows

package platform

// GetDrives returns nil on non-Windows platforms.
func GetDrives() []string {
	return nil
}

// IsRootPath returns true if path is the filesystem root.
func IsRootPath(path string) bool {
	return path == "/"
}
