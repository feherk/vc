package fnbar

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// FnBar renders the F1-F12 function key bar at the bottom of the screen.
type FnBar struct {
	*tview.Box
}

var fnLabels = [12]string{
	"Conn", "Zip", "View", "Edit", "Copy",
	"Move", "MkDir", "Del", "Menu", "Quit",
	"Select", "Name",
}

func New() *FnBar {
	return &FnBar{
		Box: tview.NewBox().SetBackgroundColor(theme.ColorFnLabelBg),
	}
}

func (f *FnBar) Draw(screen tcell.Screen) {
	f.Box.DrawForSubclass(screen, f)
	x, y, width, _ := f.GetInnerRect()

	slotWidth := width / 12

	for i := 0; i < 12; i++ {
		sx := x + i*slotWidth

		// Draw key number
		keyStr := fmt.Sprintf("%d", i+1)

		for j, ch := range keyStr {
			screen.SetContent(sx+j, y, ch, nil,
				tcell.StyleDefault.
					Foreground(theme.ColorFnKeyFg).
					Background(theme.ColorFnKeyBg))
		}

		// Draw label
		labelStart := sx + len(keyStr)
		label := fnLabels[i]
		remaining := slotWidth - len(keyStr)
		if len(label) > remaining {
			label = label[:remaining]
		}

		for j, ch := range label {
			screen.SetContent(labelStart+j, y, ch, nil,
				tcell.StyleDefault.
					Foreground(theme.ColorFnLabelFg).
					Background(theme.ColorFnLabelBg))
		}

		for j := len(label); j < remaining; j++ {
			screen.SetContent(labelStart+j, y, ' ', nil,
				tcell.StyleDefault.
					Foreground(theme.ColorFnLabelFg).
					Background(theme.ColorFnLabelBg))
		}
	}
}
