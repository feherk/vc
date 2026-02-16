package panel

import (
	"fmt"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/model"
	"github.com/feherkaroly/vc/internal/theme"
)

// RenderFull renders the panel table in Full mode (Name, Size, Date, Time).
func RenderFull(table *tview.Table, entries []model.FileEntry, cursor int, sel *model.Selection, active bool) {
	table.Clear()

	// Header row
	headers := []string{"Name", "Size", "Date", "Time"}
	for col, h := range headers {
		cell := tview.NewTableCell(h).
			SetTextColor(tcell.ColorYellow).
			SetBackgroundColor(theme.ColorPanelBg).
			SetSelectable(false).
			SetAttributes(tcell.AttrBold)
		if col == 0 {
			cell.SetExpansion(1)
		}
		table.SetCell(0, col, cell)
	}

	for i, entry := range entries {
		row := i + 1 // +1 for header

		fg := fileColor(entry)
		bg := theme.ColorPanelBg

		if sel.IsSelected(entry.Name) {
			fg = theme.ColorSelected
		}

		// Name column
		name := entry.Name
		nameCell := tview.NewTableCell(name).
			SetTextColor(fg).
			SetBackgroundColor(bg).
			SetExpansion(1)
		if entry.IsDir {
			nameCell.SetAttributes(tcell.AttrBold)
		}
		table.SetCell(row, 0, nameCell)

		// Size column
		sizeStr := formatSize(entry)
		sizeCell := tview.NewTableCell(sizeStr).
			SetTextColor(fg).
			SetBackgroundColor(bg).
			SetAlign(tview.AlignRight)
		table.SetCell(row, 1, sizeCell)

		// Date column
		dateCell := tview.NewTableCell(entry.ModTime.Format("02.01.06")).
			SetTextColor(fg).
			SetBackgroundColor(bg)
		table.SetCell(row, 2, dateCell)

		// Time column
		timeCell := tview.NewTableCell(entry.ModTime.Format("15:04")).
			SetTextColor(fg).
			SetBackgroundColor(bg)
		table.SetCell(row, 3, timeCell)
	}

	// Set cursor
	if len(entries) > 0 {
		table.Select(cursor+1, 0) // +1 for header row
	}
}

// RenderBrief renders the panel table in Brief mode (names in multiple columns).
func RenderBrief(table *tview.Table, entries []model.FileEntry, cursor int, sel *model.Selection, active bool, height int) {
	table.Clear()

	if height <= 0 {
		height = 20
	}

	rows := height
	cols := (len(entries) + rows - 1) / rows
	if cols < 1 {
		cols = 1
	}

	for i, entry := range entries {
		col := i / rows
		row := i % rows

		fg := fileColor(entry)
		bg := theme.ColorPanelBg

		if sel.IsSelected(entry.Name) {
			fg = theme.ColorSelected
		}

		cell := tview.NewTableCell(entry.Name).
			SetTextColor(fg).
			SetBackgroundColor(bg).
			SetExpansion(1)
		if entry.IsDir {
			cell.SetAttributes(tcell.AttrBold)
		}
		table.SetCell(row, col, cell)
	}

	// Set cursor
	if len(entries) > 0 {
		col := cursor / rows
		row := cursor % rows
		table.Select(row, col)
	}
}

func fileColor(entry model.FileEntry) tcell.Color {
	if entry.IsDir {
		return theme.ColorDirectory
	}
	if entry.Mode&0111 != 0 {
		return theme.ColorExecutable
	}
	return theme.ColorNormalFile
}

func formatSize(entry model.FileEntry) string {
	if entry.IsDir {
		if entry.DirSize >= 0 {
			return formatNumber(entry.DirSize)
		}
		return "<DIR>"
	}
	return formatNumber(entry.Size)
}

func formatNumber(n int64) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	var result strings.Builder
	for i, c := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result.WriteRune(',')
		}
		result.WriteRune(c)
	}
	return result.String()
}

func FormatSummary(entries []model.FileEntry, sel *model.Selection) string {
	var fileCount, dirCount int
	var totalSize int64
	for _, e := range entries {
		if e.Name == ".." {
			continue
		}
		if e.IsDir {
			dirCount++
		} else {
			fileCount++
			totalSize += e.Size
		}
	}

	if sel.Count() > 0 {
		return fmt.Sprintf("%d selected, %s in %d/%d files",
			sel.Count(), formatBytes(sel.TotalSize(entries)), fileCount, fileCount+dirCount)
	}

	return fmt.Sprintf("%s in %d file(s)", formatBytes(totalSize), fileCount)
}

func formatBytes(b int64) string {
	const (
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)
	switch {
	case b >= GB:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(GB))
	case b >= MB:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(MB))
	case b >= KB:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(KB))
	default:
		return fmt.Sprintf("%d bytes", b)
	}
}

// FormatTime formats a time for display.
func FormatTime(t time.Time) string {
	return t.Format("02.01.2006 15:04")
}
