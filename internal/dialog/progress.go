package dialog

import (
	"fmt"
	"strings"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/fileops"
	"github.com/feherkaroly/vc/internal/theme"
)

const progressBarWidth = 30

// ProgressDialog displays a file operation progress with an Abort button.
type ProgressDialog struct {
	*tview.Flex
	text   *tview.TextView
	button *tview.Button
	onDone func()
}

// NewProgressDialog creates a progress dialog with the given title (e.g. "Copying", "Moving").
func NewProgressDialog(title string, onAbort func()) *ProgressDialog {
	d := &ProgressDialog{}

	d.text = tview.NewTextView()
	d.text.SetDynamicColors(false)
	d.text.SetTextAlign(tview.AlignCenter)
	d.text.SetBackgroundColor(theme.ColorDialogBg)
	d.text.SetTextColor(theme.ColorDialogFg)

	d.button = tview.NewButton("Abort")
	d.button.SetBackgroundColor(theme.ColorButtonBg)
	d.button.SetLabelColor(theme.ColorButtonFg)
	d.button.SetSelectedFunc(func() {
		if onAbort != nil {
			onAbort()
		}
	})

	// Inner layout: text + centered button
	buttonRow := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(nil, 0, 1, false).
		AddItem(d.button, 10, 0, true).
		AddItem(nil, 0, 1, false)
	buttonRow.SetBackgroundColor(theme.ColorDialogBg)

	inner := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(d.text, 5, 0, false).
		AddItem(buttonRow, 1, 0, true)
	inner.SetBackgroundColor(theme.ColorDialogBg)
	inner.SetBorder(true)
	inner.SetBorderColor(theme.ColorDialogBorder)
	inner.SetTitle(" " + title + " ")
	inner.SetTitleColor(theme.ColorDialogFg)

	inner.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			if onAbort != nil {
				onAbort()
			}
			return nil
		}
		return event
	})

	// Center the dialog on screen
	d.Flex = tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(inner, 40, 0, true).
			AddItem(nil, 0, 1, false),
			9, 0, true).
		AddItem(nil, 0, 1, false)

	d.Update(fileops.Progress{})
	return d
}

// Update refreshes the dialog with new progress data.
func (d *ProgressDialog) Update(p fileops.Progress) {
	pct := p.Percent()
	bar := buildProgressBar(pct, progressBarWidth)

	var lines []string
	lines = append(lines, "")

	// File name
	name := p.FileName
	if name == "" {
		name = "..."
	}
	lines = append(lines, name)

	// Progress bar + percentage
	lines = append(lines, fmt.Sprintf("%s %3d%%", bar, pct))

	// Byte counters
	lines = append(lines, fmt.Sprintf("%s / %s", formatSize(p.Done), formatSize(p.Total)))

	// File counter
	if p.FileCount > 0 {
		lines = append(lines, fmt.Sprintf("File %d of %d", p.FileIndex, p.FileCount))
	}

	d.text.SetText(strings.Join(lines, "\n"))
}

// buildProgressBar creates an ASCII progress bar like [████████░░░░░░░]
func buildProgressBar(percent, width int) string {
	if percent < 0 {
		percent = 0
	}
	if percent > 100 {
		percent = 100
	}
	filled := width * percent / 100
	empty := width - filled
	return "[" + strings.Repeat("\u2588", filled) + strings.Repeat("\u2591", empty) + "]"
}

// formatSize formats bytes into a human-readable string.
func formatSize(b int64) string {
	switch {
	case b >= 1<<30:
		return fmt.Sprintf("%.1f GB", float64(b)/float64(1<<30))
	case b >= 1<<20:
		return fmt.Sprintf("%.1f MB", float64(b)/float64(1<<20))
	case b >= 1<<10:
		return fmt.Sprintf("%.1f KB", float64(b)/float64(1<<10))
	default:
		return fmt.Sprintf("%d B", b)
	}
}
