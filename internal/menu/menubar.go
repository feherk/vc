package menu

import (
	"unicode"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

type MenuBar struct {
	*tview.Box
	Items    []string
	HotKeys  []rune
	Selected int
	Active   bool
	Version  string
	OnClick  func(idx int)
}

func NewMenuBar() *MenuBar {
	m := &MenuBar{
		Box:     tview.NewBox().SetBackgroundColor(theme.ColorMenuBarBg),
		Items:   []string{" Left ", " File ", " Commands ", " Right "},
		HotKeys: []rune{'L', 'F', 'C', 'R'},
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

		// Find hotkey position in item string
		hotKeyPos := -1
		if i < len(m.HotKeys) && m.HotKeys[i] != 0 {
			upper := unicode.ToUpper(m.HotKeys[i])
			for k, ch := range item {
				if unicode.ToUpper(ch) == upper {
					hotKeyPos = k
					break
				}
			}
		}

		j := 0
		for _, ch := range item {
			if pos < x+width {
				charFg := fg
				if j == hotKeyPos {
					charFg = theme.ColorMenuHotKey
				}
				screen.SetContent(pos, y, ch, nil,
					tcell.StyleDefault.Foreground(charFg).Background(bg))
				pos++
			}
			j++
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

// FindItemByHotKey returns the index of the menu bar item whose HotKey matches r (case-insensitive).
// Returns -1 if no match is found.
func (m *MenuBar) FindItemByHotKey(r rune) int {
	upper := unicode.ToUpper(r)
	for i, hk := range m.HotKeys {
		if unicode.ToUpper(hk) == upper {
			return i
		}
	}
	return -1
}

// MouseHandler returns the mouse handler for the menu bar.
func (m *MenuBar) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return m.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		if action != tview.MouseLeftClick {
			return false, nil
		}
		mx, my := event.Position()
		x, y, _, _ := m.GetInnerRect()
		if my != y {
			return false, nil
		}
		pos := x
		for i, item := range m.Items {
			end := pos + len(item)
			if mx >= pos && mx < end {
				if m.OnClick != nil {
					m.OnClick(i)
				}
				return true, nil
			}
			pos = end
		}
		return false, nil
	})
}
