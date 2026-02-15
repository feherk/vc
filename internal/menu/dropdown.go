package menu

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

type MenuItem struct {
	Label  string
	Key    string
	Action func()
	IsSep  bool
}

type Dropdown struct {
	*tview.Box
	Items    []MenuItem
	Selected int
	Visible  bool
	X, Y     int
}

func NewDropdown() *Dropdown {
	return &Dropdown{
		Box: tview.NewBox().SetBackgroundColor(theme.ColorDialogBg),
	}
}

func (d *Dropdown) SetItems(items []MenuItem) {
	d.Items = items
	d.Selected = 0
	for d.Selected < len(d.Items) && d.Items[d.Selected].IsSep {
		d.Selected++
	}
}

func (d *Dropdown) MoveDown() {
	for {
		d.Selected++
		if d.Selected >= len(d.Items) {
			d.Selected = 0
		}
		if !d.Items[d.Selected].IsSep {
			break
		}
	}
}

func (d *Dropdown) MoveUp() {
	for {
		d.Selected--
		if d.Selected < 0 {
			d.Selected = len(d.Items) - 1
		}
		if !d.Items[d.Selected].IsSep {
			break
		}
	}
}

func (d *Dropdown) Draw(screen tcell.Screen) {
	if !d.Visible {
		return
	}

	maxWidth := 0
	for _, item := range d.Items {
		w := len(item.Label) + len(item.Key) + 4
		if w > maxWidth {
			maxWidth = w
		}
	}

	height := len(d.Items) + 2
	width := maxWidth + 2

	x, y := d.X, d.Y

	for row := 0; row < height; row++ {
		for col := 0; col < width; col++ {
			ch := ' '
			fg := theme.ColorDialogFg
			bg := theme.ColorDialogBg

			if row == 0 || row == height-1 {
				if col == 0 {
					if row == 0 {
						ch = '\u250c'
					} else {
						ch = '\u2514'
					}
				} else if col == width-1 {
					if row == 0 {
						ch = '\u2510'
					} else {
						ch = '\u2518'
					}
				} else {
					ch = '\u2500'
				}
				fg = theme.ColorDialogBorder
			} else if col == 0 || col == width-1 {
				ch = '\u2502'
				fg = theme.ColorDialogBorder
			}

			screen.SetContent(x+col, y+row, ch, nil,
				tcell.StyleDefault.Foreground(fg).Background(bg))
		}
	}

	for i, item := range d.Items {
		iy := y + 1 + i
		ix := x + 1

		fg := theme.ColorDialogFg
		bg := theme.ColorDialogBg
		if i == d.Selected {
			fg = theme.ColorDialogBg
			bg = theme.ColorDialogFg
		}

		if item.IsSep {
			for col := 0; col < width-2; col++ {
				screen.SetContent(ix+col, iy, '\u2500', nil,
					tcell.StyleDefault.Foreground(theme.ColorDialogBorder).Background(theme.ColorDialogBg))
			}
			continue
		}

		for col := 0; col < width-2; col++ {
			screen.SetContent(ix+col, iy, ' ', nil,
				tcell.StyleDefault.Foreground(fg).Background(bg))
		}

		for j, ch := range item.Label {
			if j < width-2 {
				screen.SetContent(ix+1+j, iy, ch, nil,
					tcell.StyleDefault.Foreground(fg).Background(bg))
			}
		}

		if item.Key != "" {
			kx := ix + width - 3 - len(item.Key)
			for j, ch := range item.Key {
				screen.SetContent(kx+j, iy, ch, nil,
					tcell.StyleDefault.Foreground(fg).Background(bg))
			}
		}
	}
}

func (d *Dropdown) CurrentAction() func() {
	if d.Selected >= 0 && d.Selected < len(d.Items) {
		return d.Items[d.Selected].Action
	}
	return nil
}
