package dialog

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// formatBox is a custom widget that draws format options directly to screen.
type formatBox struct {
	*tview.Box
	formats  []string
	selected int
	callback func(string)
	onCancel func()
}

func (f *formatBox) Draw(screen tcell.Screen) {
	x, y, width, _ := f.GetInnerRect()

	// Center the box
	boxW := 20
	boxH := len(f.formats) + 2
	bx := x + (width-boxW)/2
	_, _, _, totalH := f.Box.GetRect()
	by := y + (totalH-boxH)/2

	// Draw border
	for row := 0; row < boxH; row++ {
		for col := 0; col < boxW; col++ {
			ch := ' '
			fg := theme.ColorDialogBorder
			bg := theme.ColorDialogBg

			if row == 0 || row == boxH-1 {
				if col == 0 {
					if row == 0 {
						ch = '┌'
					} else {
						ch = '└'
					}
				} else if col == boxW-1 {
					if row == 0 {
						ch = '┐'
					} else {
						ch = '┘'
					}
				} else {
					ch = '─'
				}
			} else if col == 0 || col == boxW-1 {
				ch = '│'
			} else {
				fg = theme.ColorDialogFg
			}

			screen.SetContent(bx+col, by+row, ch, nil,
				tcell.StyleDefault.Foreground(fg).Background(bg))
		}
	}

	// Draw title
	title := " Format "
	tx := bx + (boxW-len(title))/2
	for i, ch := range title {
		screen.SetContent(tx+i, by, ch, nil,
			tcell.StyleDefault.Foreground(theme.ColorHeaderFg).Background(theme.ColorDialogBg))
	}

	// Draw items
	for i, format := range f.formats {
		iy := by + 1 + i
		fg := theme.ColorDialogFg
		bg := theme.ColorDialogBg
		if i == f.selected {
			fg = theme.ColorDialogBg
			bg = theme.ColorDialogFg
		}

		for col := 1; col < boxW-1; col++ {
			screen.SetContent(bx+col, iy, ' ', nil,
				tcell.StyleDefault.Foreground(fg).Background(bg))
		}

		label := format
		for j, ch := range label {
			if j < boxW-4 {
				screen.SetContent(bx+2+j, iy, ch, nil,
					tcell.StyleDefault.Foreground(fg).Background(bg))
			}
		}
	}
}

func (f *formatBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return f.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyDown:
			f.selected++
			if f.selected >= len(f.formats) {
				f.selected = 0
			}
		case tcell.KeyUp:
			f.selected--
			if f.selected < 0 {
				f.selected = len(f.formats) - 1
			}
		case tcell.KeyEnter:
			f.callback(f.formats[f.selected])
		case tcell.KeyEscape:
			f.onCancel()
		case tcell.KeyRune:
			idx := int(event.Rune() - '1')
			if idx >= 0 && idx < len(f.formats) {
				f.callback(f.formats[idx])
			}
		}
	})
}

func (f *formatBox) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return f.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		if action != tview.MouseLeftClick {
			return false, nil
		}
		mx, my := event.Position()
		x, y, width, _ := f.GetInnerRect()

		boxW := 20
		boxH := len(f.formats) + 2
		bx := x + (width-boxW)/2
		_, _, _, totalH := f.Box.GetRect()
		by := y + (totalH-boxH)/2

		// Check if click is inside the box content area
		row := my - by - 1
		if mx > bx && mx < bx+boxW-1 && row >= 0 && row < len(f.formats) {
			f.callback(f.formats[row])
			return true, nil
		}

		// Click outside → cancel
		if mx < bx || mx >= bx+boxW || my < by || my >= by+boxH {
			f.onCancel()
			return true, nil
		}

		return true, nil
	})
}

// ShowFormatDialog displays a format selection dialog for compression.
func ShowFormatDialog(pages *tview.Pages, singleFile bool, isEnc bool, isArchive bool, callback func(format string), onCancel func()) {
	var formats []string
	if singleFile && isArchive {
		formats = []string{"extract", "encrypt"}
	} else {
		formats = []string{"zip", "tar", "tar.gz"}
		if singleFile && isEnc {
			formats = append(formats, "decrypt")
		} else if singleFile && !isEnc {
			formats = append(formats, "encrypt")
		}
	}

	fb := &formatBox{
		Box:      tview.NewBox(),
		formats:  formats,
		selected: 0,
		callback: callback,
		onCancel: onCancel,
	}

	pages.AddPage("format", fb, true, true)
}
