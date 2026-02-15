package fileops

// Progress represents the progress of a file operation.
type Progress struct {
	FileName  string
	Total     int64
	Done      int64
	FileIndex int // current file (1-based)
	FileCount int // total number of files
}

// Percent returns the completion percentage (0-100).
func (p Progress) Percent() int {
	if p.Total == 0 {
		return 100
	}
	return int(p.Done * 100 / p.Total)
}
