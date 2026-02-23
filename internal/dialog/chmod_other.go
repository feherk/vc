//go:build !unix

package dialog

import "fmt"

// GetFileOwnership is not supported on non-Unix platforms.
func GetFileOwnership(path string) (uid, gid int, owner, group string, err error) {
	return 0, 0, "", "", fmt.Errorf("not supported")
}

// GetDefaultACL is not supported on non-Unix platforms.
func GetDefaultACL(path string) ([3][3]bool, error) {
	return [3][3]bool{}, nil
}

// SetDefaultACL is not supported on non-Unix platforms.
func SetDefaultACL(path string, acl [3][3]bool) error {
	return nil
}

// ListUsers returns nil on non-Unix platforms.
func ListUsers() []string { return nil }

// ListGroups returns nil on non-Unix platforms.
func ListGroups() []string { return nil }

// HasACLSupport returns false on non-Unix platforms.
func HasACLSupport() bool {
	return false
}
