package viewer

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// Viewer is a full-screen file viewer (F3).
type Viewer struct {
	*tview.Frame
	textView *tview.TextView
	doneFunc func()
	filePath string
}

// New creates a new file viewer for the given path.
func New(path string) *Viewer {
	tv := tview.NewTextView().
		SetDynamicColors(false).
		SetScrollable(true).
		SetWrap(true)

	tv.SetBackgroundColor(theme.ColorPanelBg)
	tv.SetTextColor(theme.ColorNormalFile)

	frame := tview.NewFrame(tv).
		SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBorderColor(theme.ColorActiveBorder)
	frame.SetBackgroundColor(theme.ColorPanelBg)
	frame.SetTitle(" " + path + " ")
	frame.SetTitleColor(theme.ColorHeaderFg)

	v := &Viewer{
		Frame:    frame,
		textView: tv,
		filePath: path,
	}

	data, err := os.ReadFile(path)
	if err != nil {
		tv.SetText(fmt.Sprintf("Error reading file: %v", err))
	} else {
		tv.SetText(string(data))
	}

	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyF3, tcell.KeyF10:
			if v.doneFunc != nil {
				v.doneFunc()
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' || event.Rune() == 'Q' {
				if v.doneFunc != nil {
					v.doneFunc()
				}
				return nil
			}
		}
		return event
	})

	return v
}

// NewFromText creates a viewer that displays the given text content.
func NewFromText(title string, content string) *Viewer {
	tv := tview.NewTextView().
		SetDynamicColors(false).
		SetScrollable(true).
		SetWrap(false)

	tv.SetBackgroundColor(theme.ColorPanelBg)
	tv.SetTextColor(theme.ColorNormalFile)
	tv.SetText(content)

	frame := tview.NewFrame(tv).
		SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBorderColor(theme.ColorActiveBorder)
	frame.SetBackgroundColor(theme.ColorPanelBg)
	frame.SetTitle(" " + title + " ")
	frame.SetTitleColor(theme.ColorHeaderFg)

	v := &Viewer{
		Frame:    frame,
		textView: tv,
		filePath: title,
	}

	tv.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape, tcell.KeyF3, tcell.KeyF10:
			if v.doneFunc != nil {
				v.doneFunc()
			}
			return nil
		case tcell.KeyRune:
			if event.Rune() == 'q' || event.Rune() == 'Q' {
				if v.doneFunc != nil {
					v.doneFunc()
				}
				return nil
			}
		}
		return event
	})

	return v
}

func (v *Viewer) SetDoneFunc(f func()) {
	v.doneFunc = f
}

func (v *Viewer) Focus(delegate func(p tview.Primitive)) {
	delegate(v.textView)
}
