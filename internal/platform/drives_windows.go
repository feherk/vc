//go:build windows

package platform

import (
	"golang.org/x/sys/windows"
)

// GetDrives returns a list of available drive roots (e.g. "A:\", "C:\").
func GetDrives() []string {
	mask, err := windows.GetLogicalDrives()
	if err != nil {
		return nil
	}

	var drives []string
	for i := 0; i < 26; i++ {
		if mask&(1<<uint(i)) != 0 {
			letter := string(rune('A' + i))
			drives = append(drives, letter+`:\`)
		}
	}
	return drives
}

// IsRootPath returns true if the path is a drive root like "C:\".
func IsRootPath(path string) bool {
	if len(path) == 3 && path[1] == ':' && (path[2] == '\\' || path[2] == '/') {
		c := path[0]
		return (c >= 'A' && c <= 'Z') || (c >= 'a' && c <= 'z')
	}
	return false
}
