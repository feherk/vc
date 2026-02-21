package panel

import (
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/model"
	"github.com/feherkaroly/vc/internal/platform"
	"github.com/feherkaroly/vc/internal/theme"
	"github.com/feherkaroly/vc/internal/vfs"
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

	FS              vfs.FileSystem
	ConnectedServer string
}

// NewPanel creates a new file panel at the given path.
func NewPanel(path string, fs vfs.FileSystem) *Panel {
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
		FS:        fs,
	}

	p.LoadDir()
	return p
}

// loadEntries reads the current directory and populates entries (no rendering).
func (p *Panel) loadEntries() error {
	dirEntries, err := p.FS.ReadDir(p.Path)
	if err != nil {
		return err
	}

	p.Entries = make([]model.FileEntry, 0, len(dirEntries)+1)

	// Add parent directory entry if not root
	if !platform.IsRootPath(p.Path) {
		p.Entries = append(p.Entries, model.FileEntry{
			Name:    "..",
			IsDir:   true,
			DirSize: -1,
		})
	}

	for _, de := range dirEntries {
		entry := model.FileEntry{
			Name:    de.Name,
			Size:    de.Size,
			ModTime: de.ModTime,
			Mode:    de.Mode,
			IsDir:   de.IsDir,
			IsLink:  de.IsLink,
			LinkTo:  de.LinkTo,
			DirSize: -1,
		}

		p.Entries = append(p.Entries, entry)
	}

	SortEntries(p.Entries, p.SortMode)
	return nil
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
	title := p.Path
	if p.ConnectedServer != "" {
		title = "[" + p.ConnectedServer + "] " + p.Path
	} else {
		title = shortenPath(p.Path)
	}
	p.Box.SetTitle(" " + title + " ")
	p.Box.SetTitleAlign(tview.AlignLeft)
	p.Box.SetTitleColor(theme.ColorHeaderFg)

	p.Box.Clear()
	if e := p.CurrentEntry(); e != nil && e.IsLink && e.LinkTo != "" {
		p.Box.AddText("@ → "+e.LinkTo, false, tview.AlignCenter, theme.ColorSymlink)
	} else {
		summary := FormatSummary(p.Entries, p.Selection)
		p.Box.AddText(summary, false, tview.AlignCenter, theme.ColorHeaderFg)
	}
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
	return p.FS.Join(p.Path, e.Name)
}

// Enter handles Enter key: navigate into directory or return entry.
// Returns the file entry (if a file was selected) and whether a drive switch is needed.
func (p *Panel) Enter() (*model.FileEntry, bool) {
	e := p.CurrentEntry()
	if e == nil {
		return nil, false
	}

	if e.IsDir {
		if e.Name == ".." {
			atRoot := p.GoParent()
			return nil, atRoot
		}
		newPath := p.FS.Join(p.Path, e.Name)
		p.NavigateTo(newPath, "")
		return nil, false
	}

	return e, false
}

// GoParent navigates to the parent directory.
// Returns true if already at root (caller should show drive dialog).
func (p *Panel) GoParent() bool {
	parent := p.FS.Dir(p.Path)
	if parent == p.Path {
		return true
	}
	oldName := p.FS.Base(p.Path)
	p.NavigateTo(parent, oldName)
	return false
}

// NavigateTo changes to a new directory, optionally focusing on a named entry.
func (p *Panel) NavigateTo(newPath string, focusName string) {
	oldPath := p.Path
	oldEntries := p.Entries
	oldCursor := p.Cursor

	p.Path = newPath
	p.Selection.Clear()
	p.Cursor = 0

	if err := p.loadEntries(); err != nil {
		// Restore previous state on error
		p.Path = oldPath
		p.Entries = oldEntries
		p.Cursor = oldCursor
		return
	}

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


// ToggleSelectionAt toggles selection of the entry at the given index without moving cursor.
func (p *Panel) ToggleSelectionAt(idx int) {
	if idx < 0 || idx >= len(p.Entries) {
		return
	}
	e := &p.Entries[idx]
	if e.Name == ".." {
		return
	}
	p.Selection.Toggle(e.Name)
	p.Render()
	p.UpdateTitle()
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
	p.UpdateTitle()
}

// IsRemote returns true if the panel is connected to a remote server.
func (p *Panel) IsRemote() bool {
	return !p.FS.IsLocal()
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && len(path) > len(home) && path[:len(home)] == home {
		return "~" + path[len(home):]
	}
	return path
}
