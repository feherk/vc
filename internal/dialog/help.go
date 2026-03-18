package dialog

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

type helpBox struct {
	*tview.Box
	lines   []string
	scroll  int
	onClose func()
}

func newHelpBox(onClose func()) *helpBox {
	h := &helpBox{
		Box:     tview.NewBox(),
		onClose: onClose,
		lines: []string{
			" Navigation",
			" ──────────────────────────────────",
			" Tab            Switch panel",
			" Enter          Open dir / file",
			" Backspace      Parent directory",
			" Ctrl+R         Refresh panels",
			" Right arrow    Next column (Brief)",
			"",
			" File Operations",
			" ──────────────────────────────────",
			" F2             Archive / Encrypt",
			" F3             View file",
			" F4             Edit file",
			" F5             Copy",
			" F6             Move / Rename",
			" F7             Create directory",
			" F8 / Del       Delete",
			" A              File attributes",
			"",
			" Selection",
			" ──────────────────────────────────",
			" Ctrl+S / Ins   Toggle selection",
			" Space          Calculate dir size",
			"",
			" Other",
			" ──────────────────────────────────",
			" Ctrl+N         Quick paths",
			" F9             Menu",
			" F10            Quit",
			"",
			" Mouse",
			" ──────────────────────────────────",
			" Left click     Position cursor",
			" Double-click   Open dir / file",
			" Right-click    Toggle selection",
			"",
			" Menu: Left/Right → Connect server",
			"       File → operations",
			"       Commands → tools, config",
		},
	}
	h.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyF1, tcell.KeyEnter:
			if h.onClose != nil {
				h.onClose()
			}
			return nil
		case tcell.KeyDown:
			maxScroll := len(h.lines) - 20
			if maxScroll < 0 {
				maxScroll = 0
			}
			if h.scroll < maxScroll {
				h.scroll++
			}
			return nil
		case tcell.KeyUp:
			if h.scroll > 0 {
				h.scroll--
			}
			return nil
		}
		return event
	})
	return h
}

func (h *helpBox) Draw(screen tcell.Screen) {
	screenW, screenH := screen.Size()

	dialogW := 42
	dialogH := 24
	if dialogH > screenH-2 {
		dialogH = screenH - 2
	}
	x0 := (screenW - dialogW) / 2
	y0 := (screenH - dialogH) / 2

	h.SetRect(x0, y0, dialogW, dialogH)

	// Draw background
	bgStyle := tcell.StyleDefault.Foreground(theme.ColorDialogFg).Background(theme.ColorDialogBg)
	for row := y0; row < y0+dialogH; row++ {
		for col := x0; col < x0+dialogW; col++ {
			screen.SetContent(col, row, ' ', nil, bgStyle)
		}
	}

	// Draw border
	borderStyle := tcell.StyleDefault.Foreground(theme.ColorDialogBorder).Background(theme.ColorDialogBg)
	screen.SetContent(x0, y0, '┌', nil, borderStyle)
	screen.SetContent(x0+dialogW-1, y0, '┐', nil, borderStyle)
	screen.SetContent(x0, y0+dialogH-1, '└', nil, borderStyle)
	screen.SetContent(x0+dialogW-1, y0+dialogH-1, '┘', nil, borderStyle)
	for col := x0 + 1; col < x0+dialogW-1; col++ {
		screen.SetContent(col, y0, '─', nil, borderStyle)
		screen.SetContent(col, y0+dialogH-1, '─', nil, borderStyle)
	}
	for row := y0 + 1; row < y0+dialogH-1; row++ {
		screen.SetContent(x0, row, '│', nil, borderStyle)
		screen.SetContent(x0+dialogW-1, row, '│', nil, borderStyle)
	}

	// Title
	title := " Help "
	titleStyle := tcell.StyleDefault.Foreground(theme.ColorHeaderFg).Background(theme.ColorDialogBg)
	tx := x0 + (dialogW-len(title))/2
	for i, ch := range title {
		screen.SetContent(tx+i, y0, ch, nil, titleStyle)
	}

	// Content area
	contentH := dialogH - 2
	for i := 0; i < contentH && h.scroll+i < len(h.lines); i++ {
		line := h.lines[h.scroll+i]
		col := x0 + 1
		for _, ch := range line {
			if col >= x0+dialogW-1 {
				break
			}
			screen.SetContent(col, y0+1+i, ch, nil, bgStyle)
			col++
		}
	}

	// Footer hint
	hint := " Esc/Enter - Close "
	hintStyle := tcell.StyleDefault.Foreground(tcell.ColorYellow).Background(theme.ColorDialogBg)
	hx := x0 + (dialogW-len(hint))/2
	for i, ch := range hint {
		screen.SetContent(hx+i, y0+dialogH-1, ch, nil, hintStyle)
	}
}

// ShowHelp displays the help dialog with keyboard shortcuts.
func ShowHelp(pages *tview.Pages, onClose func()) {
	box := newHelpBox(onClose)
	pages.AddPage("help_dialog", box, true, true)
}
