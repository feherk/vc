package model

// Selection tracks multi-selected files via Insert key.
type Selection struct {
	items map[string]bool
}

func NewSelection() *Selection {
	return &Selection{items: make(map[string]bool)}
}

func (s *Selection) Toggle(name string) {
	if s.items[name] {
		delete(s.items, name)
	} else {
		s.items[name] = true
	}
}

func (s *Selection) IsSelected(name string) bool {
	return s.items[name]
}

func (s *Selection) Clear() {
	s.items = make(map[string]bool)
}

func (s *Selection) Count() int {
	return len(s.items)
}

func (s *Selection) Items() []string {
	result := make([]string, 0, len(s.items))
	for k := range s.items {
		result = append(result, k)
	}
	return result
}

func (s *Selection) TotalSize(entries []FileEntry) int64 {
	var total int64
	for _, e := range entries {
		if s.items[e.Name] {
			if e.IsDir {
				if e.DirSize > 0 {
					total += e.DirSize
				}
			} else {
				total += e.Size
			}
		}
	}
	return total
}
