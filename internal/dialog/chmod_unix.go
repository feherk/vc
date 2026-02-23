//go:build unix

package dialog

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/user"
	"sort"
	"strconv"
	"strings"
	"syscall"
)

// GetFileOwnership returns uid, gid, owner name and group name for a file.
func GetFileOwnership(path string) (uid, gid int, owner, group string, err error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return 0, 0, "", "", err
	}
	stat, ok := fi.Sys().(*syscall.Stat_t)
	if !ok {
		return 0, 0, "", "", fmt.Errorf("cannot get file ownership")
	}
	uid = int(stat.Uid)
	gid = int(stat.Gid)
	if u, err := user.LookupId(strconv.Itoa(uid)); err == nil {
		owner = u.Username
	} else {
		owner = strconv.Itoa(uid)
	}
	if g, err := user.LookupGroupId(strconv.Itoa(gid)); err == nil {
		group = g.Name
	} else {
		group = strconv.Itoa(gid)
	}
	return uid, gid, owner, group, nil
}

// GetDefaultACL reads the default ACL of a directory using getfacl.
func GetDefaultACL(path string) ([3][3]bool, error) {
	var acl [3][3]bool
	cmd := exec.Command("getfacl", "-p", path)
	out, err := cmd.Output()
	if err != nil {
		return acl, err
	}
	scanner := bufio.NewScanner(strings.NewReader(string(out)))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "default:user::") {
			parseACLBits(strings.TrimPrefix(line, "default:user::"), &acl[0])
		} else if strings.HasPrefix(line, "default:group::") {
			parseACLBits(strings.TrimPrefix(line, "default:group::"), &acl[1])
		} else if strings.HasPrefix(line, "default:other::") {
			parseACLBits(strings.TrimPrefix(line, "default:other::"), &acl[2])
		}
	}
	return acl, nil
}

func parseACLBits(s string, bits *[3]bool) {
	if len(s) >= 3 {
		bits[0] = s[0] == 'r'
		bits[1] = s[1] == 'w'
		bits[2] = s[2] == 'x'
	}
}

// SetDefaultACL sets the default ACL on a directory using setfacl.
func SetDefaultACL(path string, acl [3][3]bool) error {
	spec := fmt.Sprintf("u::%s,g::%s,o::%s",
		aclBitsToString(acl[0]),
		aclBitsToString(acl[1]),
		aclBitsToString(acl[2]))
	cmd := exec.Command("setfacl", "-d", "-m", spec, path)
	return cmd.Run()
}

func aclBitsToString(bits [3]bool) string {
	var b [3]byte
	if bits[0] {
		b[0] = 'r'
	} else {
		b[0] = '-'
	}
	if bits[1] {
		b[1] = 'w'
	} else {
		b[1] = '-'
	}
	if bits[2] {
		b[2] = 'x'
	} else {
		b[2] = '-'
	}
	return string(b[:])
}

// ListUsers returns a sorted list of usernames from /etc/passwd.
func ListUsers() []string {
	f, err := os.Open("/etc/passwd")
	if err != nil {
		return nil
	}
	defer f.Close()
	var users []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 0 && parts[0] != "" {
			users = append(users, parts[0])
		}
	}
	sort.Strings(users)
	return users
}

// ListGroups returns a sorted list of group names from /etc/group.
func ListGroups() []string {
	f, err := os.Open("/etc/group")
	if err != nil {
		return nil
	}
	defer f.Close()
	var groups []string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if line == "" || line[0] == '#' {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) > 0 && parts[0] != "" {
			groups = append(groups, parts[0])
		}
	}
	sort.Strings(groups)
	return groups
}

// HasACLSupport returns true if getfacl is available.
// Default ACL support requires the "acl" package on Linux: sudo apt install acl
// This provides the getfacl/setfacl commands used for reading and setting default ACLs.
func HasACLSupport() bool {
	_, err := exec.LookPath("getfacl")
	return err == nil
}
