package panel

import (
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/model"
	"github.com/feherkaroly/vc/internal/theme"
)

// Panel represents one side of the dual-pane file manager.
type Panel struct {
	Table     *tview.Table
	Box       *tview.Frame
	Path      string
	Entries   []model.FileEntry
	Cursor    int
	Selection *model.Selection
	SortMode  SortMode
	Mode      DisplayMode
	Active    bool

	SearchBuf string
}

// NewPanel creates a new file panel at the given path.
func NewPanel(path string) *Panel {
	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	table := tview.NewTable().
		SetSelectable(true, false).
		SetSelectedStyle(tcell.StyleDefault.
			Foreground(theme.ColorCursorFg).
			Background(theme.ColorCursorBg))

	table.SetBackgroundColor(theme.ColorPanelBg)
	table.SetBorderPadding(0, 0, 1, 1)
	table.SetFixed(1, 0) // Header row is non-selectable

	frame := tview.NewFrame(table).
		SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBackgroundColor(theme.ColorPanelBg)
	frame.SetBorderColor(theme.ColorInactiveBorder)

	p := &Panel{
		Table:     table,
		Box:       frame,
		Path:      absPath,
		Selection: model.NewSelection(),
		SortMode:  SortByName,
		Mode:      ModeFull,
	}

	p.LoadDir()
	return p
}

// loadEntries reads the current directory and populates entries (no rendering).
func (p *Panel) loadEntries() {
	dirEntries, err := os.ReadDir(p.Path)
	if err != nil {
		return
	}

	p.Entries = make([]model.FileEntry, 0, len(dirEntries)+1)

	// Add parent directory entry if not root
	if p.Path != "/" {
		p.Entries = append(p.Entries, model.FileEntry{
			Name:    "..",
			IsDir:   true,
			DirSize: -1,
		})
	}

	for _, de := range dirEntries {
		info, err := de.Info()
		if err != nil {
			continue
		}

		entry := model.FileEntry{
			Name:    de.Name(),
			Size:    info.Size(),
			ModTime: info.ModTime(),
			Mode:    info.Mode(),
			IsDir:   de.IsDir(),
			DirSize: -1,
		}

		// Check for symlinks
		if de.Type()&os.ModeSymlink != 0 {
			entry.IsLink = true
			if target, err := os.Readlink(filepath.Join(p.Path, de.Name())); err == nil {
				entry.LinkTo = target
			}
			if fi, err := os.Stat(filepath.Join(p.Path, de.Name())); err == nil {
				entry.IsDir = fi.IsDir()
				entry.Size = fi.Size()
				entry.ModTime = fi.ModTime()
				entry.Mode = fi.Mode()
			}
		}

		p.Entries = append(p.Entries, entry)
	}

	SortEntries(p.Entries, p.SortMode)
}

// LoadDir reads the current directory, populates entries, and renders.
func (p *Panel) LoadDir() {
	p.loadEntries()
	p.Render()
	p.UpdateTitle()
}

// Render redraws the table content.
func (p *Panel) Render() {
	switch p.Mode {
	case ModeBrief:
		p.Table.SetFixed(0, 0)
		p.Table.SetSelectable(true, true) // cell-level selection
		_, _, _, h := p.Table.GetInnerRect()
		RenderBrief(p.Table, p.Entries, p.Cursor, p.Selection, p.Active, h)
	default:
		p.Table.SetFixed(1, 0)
		p.Table.SetSelectable(true, false) // row-level selection
		RenderFull(p.Table, p.Entries, p.Cursor, p.Selection, p.Active)
	}
}

// UpdateTitle sets the panel border title to the current path.
func (p *Panel) UpdateTitle() {
	p.Box.SetTitle(" " + shortenPath(p.Path) + " ")
	p.Box.SetTitleAlign(tview.AlignLeft)
	p.Box.SetTitleColor(theme.ColorHeaderFg)

	summary := FormatSummary(p.Entries, p.Selection)
	p.Box.Clear()
	p.Box.AddText(summary, false, tview.AlignCenter, theme.ColorHeaderFg)
}

// SetActive toggles the panel's active state (border color).
func (p *Panel) SetActive(active bool) {
	p.Active = active
	if active {
		p.Box.SetBorderColor(theme.ColorActiveBorder)
	} else {
		p.Box.SetBorderColor(theme.ColorInactiveBorder)
	}
	p.Render()
}

// CurrentEntry returns the entry under the cursor.
func (p *Panel) CurrentEntry() *model.FileEntry {
	if p.Cursor < 0 || p.Cursor >= len(p.Entries) {
		return nil
	}
	return &p.Entries[p.Cursor]
}

// CurrentPath returns the full path of the entry under cursor.
func (p *Panel) CurrentPath() string {
	e := p.CurrentEntry()
	if e == nil {
		return p.Path
	}
	return filepath.Join(p.Path, e.Name)
}

// Enter handles Enter key: navigate into directory or return entry.
func (p *Panel) Enter() *model.FileEntry {
	e := p.CurrentEntry()
	if e == nil {
		return nil
	}

	if e.IsDir {
		if e.Name == ".." {
			p.GoParent()
		} else {
			newPath := filepath.Join(p.Path, e.Name)
			p.NavigateTo(newPath, "")
		}
		return nil
	}

	return e
}

// GoParent navigates to the parent directory.
func (p *Panel) GoParent() {
	parent := filepath.Dir(p.Path)
	if parent == p.Path {
		return
	}
	oldName := filepath.Base(p.Path)
	p.NavigateTo(parent, oldName)
}

// NavigateTo changes to a new directory, optionally focusing on a named entry.
func (p *Panel) NavigateTo(newPath string, focusName string) {
	p.Path = newPath
	p.Selection.Clear()
	p.Cursor = 0
	p.loadEntries()

	if focusName != "" {
		for i, e := range p.Entries {
			if e.Name == focusName {
				p.Cursor = i
				break
			}
		}
	}
	p.Render()
	p.UpdateTitle()
}

// MoveCursor moves the cursor by delta, clamping to bounds.
func (p *Panel) MoveCursor(delta int) {
	p.Cursor += delta
	if p.Cursor < 0 {
		p.Cursor = 0
	}
	if p.Cursor >= len(p.Entries) {
		p.Cursor = len(p.Entries) - 1
	}
	p.Render()
}

// SetCursor sets cursor to exact position.
func (p *Panel) SetCursor(pos int) {
	if pos < 0 {
		pos = 0
	}
	if pos >= len(p.Entries) {
		pos = len(p.Entries) - 1
	}
	p.Cursor = pos
	p.Render()
}

// SelectAndMove toggles selection of the current entry and moves cursor by delta.
func (p *Panel) SelectAndMove(delta int) {
	e := p.CurrentEntry()
	if e == nil || e.Name == ".." {
		return
	}
	p.Selection.Toggle(e.Name)
	p.MoveCursor(delta)
	p.UpdateTitle()
}

// ToggleSelection toggles selection of current entry and moves cursor down.
func (p *Panel) ToggleSelection() {
	p.SelectAndMove(1)
}

// GetSelectedOrCurrent returns selected files, or current file if nothing selected.
func (p *Panel) GetSelectedOrCurrent() []model.FileEntry {
	if p.Selection.Count() > 0 {
		var result []model.FileEntry
		selected := p.Selection.Items()
		for _, e := range p.Entries {
			for _, s := range selected {
				if e.Name == s {
					result = append(result, e)
					break
				}
			}
		}
		return result
	}

	e := p.CurrentEntry()
	if e == nil || e.Name == ".." {
		return nil
	}
	return []model.FileEntry{*e}
}

// Refresh reloads the current directory.
func (p *Panel) Refresh() {
	oldCursor := p.Cursor
	p.loadEntries()
	p.Cursor = oldCursor
	if p.Cursor >= len(p.Entries) {
		p.Cursor = len(p.Entries) - 1
	}
	if p.Cursor < 0 {
		p.Cursor = 0
	}
	p.Render()
	p.UpdateTitle()
}

// HandleSelectionChanged is called when the table selection changes.
func (p *Panel) HandleSelectionChanged(row, col int) {
	if p.Mode == ModeBrief {
		_, _, _, h := p.Table.GetInnerRect()
		if h <= 0 {
			h = 20
		}
		p.Cursor = col*h + row
	} else {
		p.Cursor = row - 1 // -1 for header
	}
	if p.Cursor < 0 {
		p.Cursor = 0
	}
	if p.Cursor >= len(p.Entries) {
		p.Cursor = len(p.Entries) - 1
	}
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && len(path) > len(home) && path[:len(home)] == home {
		return "~" + path[len(home):]
	}
	return path
}
