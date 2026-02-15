package menu

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

type MenuBar struct {
	*tview.Box
	Items    []string
	Selected int
	Active   bool
	Version  string
}

func NewMenuBar() *MenuBar {
	m := &MenuBar{
		Box:   tview.NewBox().SetBackgroundColor(theme.ColorMenuBarBg),
		Items: []string{" Left ", " File ", " Commands ", " Right "},
	}
	return m
}

func (m *MenuBar) Draw(screen tcell.Screen) {
	m.Box.DrawForSubclass(screen, m)
	x, y, width, _ := m.GetInnerRect()

	for i := 0; i < width; i++ {
		screen.SetContent(x+i, y, ' ', nil,
			tcell.StyleDefault.Foreground(theme.ColorMenuBarFg).Background(theme.ColorMenuBarBg))
	}

	pos := x
	for i, item := range m.Items {
		fg := theme.ColorMenuBarFg
		bg := theme.ColorMenuBarBg
		if m.Active && i == m.Selected {
			fg = theme.ColorMenuBarBg
			bg = theme.ColorMenuBarFg
		}

		for _, ch := range item {
			if pos < x+width {
				screen.SetContent(pos, y, ch, nil,
					tcell.StyleDefault.Foreground(fg).Background(bg))
				pos++
			}
		}
	}

	// Draw version on the right side
	if m.Version != "" {
		vLabel := "vc " + m.Version + " "
		vStart := x + width - len(vLabel)
		for i, ch := range vLabel {
			if vStart+i >= pos {
				screen.SetContent(vStart+i, y, ch, nil,
					tcell.StyleDefault.Foreground(theme.ColorMenuBarFg).Background(theme.ColorMenuBarBg))
			}
		}
	}
}

func (m *MenuBar) MoveLeft() {
	m.Selected--
	if m.Selected < 0 {
		m.Selected = len(m.Items) - 1
	}
}

func (m *MenuBar) MoveRight() {
	m.Selected++
	if m.Selected >= len(m.Items) {
		m.Selected = 0
	}
}
