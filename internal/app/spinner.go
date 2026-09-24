package app

import (
	"fmt"
	"time"

	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// spinner is a small non-modal "working" box in the bottom-right corner.
// Several operations may run at once; each gets its own page and the boxes
// are stacked upwards so none of them hides or removes another.
type spinner struct {
	name  string
	label string
	view  *tview.TextView
	done  chan struct{}
}

// startSpinner shows a spinner box and returns a function that hides it.
// Both must be called on the UI goroutine (the stop function typically from
// inside QueueUpdateDraw).
func (a *App) startSpinner(displayName string) (stop func()) {
	// keep the box narrow enough for an 80-column terminal
	if r := []rune(displayName); len(r) > 36 {
		displayName = string(r[:35]) + "…"
	}

	view := tview.NewTextView()
	view.SetBackgroundColor(theme.ColorDialogBg)
	view.SetTextColor(theme.ColorDialogFg)
	view.SetBorder(true)
	view.SetBorderColor(theme.ColorDialogBorder)
	view.SetTextAlign(tview.AlignCenter)
	view.SetText(displayName + " |")

	a.spinnerSeq++
	s := &spinner{
		name:  fmt.Sprintf("spinner-%d", a.spinnerSeq),
		label: displayName,
		view:  view,
		done:  make(chan struct{}),
	}
	a.spinners = append(a.spinners, s)
	a.layoutSpinners()
	a.Pages.AddPage(s.name, view, false, true)
	a.focusActiveTable()

	go func() {
		spinChars := [4]rune{'|', '/', '-', '\\'}
		spinIdx := 0
		ticker := time.NewTicker(150 * time.Millisecond)
		defer ticker.Stop()
		for {
			select {
			case <-s.done:
				return
			case <-ticker.C:
				spinIdx = (spinIdx + 1) % 4
				ch := spinChars[spinIdx]
				a.TviewApp.QueueUpdateDraw(func() {
					view.SetText(fmt.Sprintf("%s %c", displayName, ch))
					a.layoutSpinners() // follows terminal resizes too
				})
			}
		}
	}()

	return func() {
		close(s.done)
		for i, x := range a.spinners {
			if x == s {
				a.spinners = append(a.spinners[:i], a.spinners[i+1:]...)
				break
			}
		}
		a.Pages.RemovePage(s.name)
		a.layoutSpinners()
	}
}

// layoutSpinners places the active spinner boxes above each other in the
// bottom-right corner, the oldest at the bottom.
func (a *App) layoutSpinners() {
	_, _, screenW, screenH := a.Pages.GetInnerRect()
	for i, s := range a.spinners {
		boxW := len([]rune(s.label)) + 6 // rune count: file names may be accented
		if boxW < 18 {
			boxW = 18
		}
		s.view.SetRect(screenW-boxW-1, screenH-4-3*i, boxW, 3)
	}
}
