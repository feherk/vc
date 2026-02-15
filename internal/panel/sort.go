package panel

import (
	"sort"
	"strings"

	"github.com/feherkaroly/vc/internal/model"
)

type SortMode int

const (
	SortByName SortMode = iota
	SortByExtension
	SortBySize
	SortByTime
)

// SortEntries sorts file entries. Directories always come first (except "..").
func SortEntries(entries []model.FileEntry, mode SortMode) {
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]

		// ".." is always first
		if a.Name == ".." {
			return true
		}
		if b.Name == ".." {
			return false
		}

		// Directories before files
		if a.IsDir != b.IsDir {
			return a.IsDir
		}

		switch mode {
		case SortByExtension:
			extA := extension(a.Name)
			extB := extension(b.Name)
			if extA != extB {
				return strings.ToLower(extA) < strings.ToLower(extB)
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case SortBySize:
			if a.Size != b.Size {
				return a.Size < b.Size
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		case SortByTime:
			if !a.ModTime.Equal(b.ModTime) {
				return a.ModTime.After(b.ModTime)
			}
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		default: // SortByName
			return strings.ToLower(a.Name) < strings.ToLower(b.Name)
		}
	})
}

func extension(name string) string {
	for i := len(name) - 1; i > 0; i-- {
		if name[i] == '.' {
			return name[i+1:]
		}
	}
	return ""
}
