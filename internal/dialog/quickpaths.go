package dialog

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// QuickPathsCallbacks holds all callbacks for the quick paths dialog.
type QuickPathsCallbacks struct {
	OnSet    func(slot int)                 // S: set active panel path to slot
	OnEdit   func(slot int, current string) // E/Enter: edit slot path
	OnDelete func(slot int)                 // D: delete slot
	OnGo     func(slot int)                 // G/number: navigate to slot
	OnClose  func()                         // Esc: close
}

// ShowQuickPathsDialog displays the quick paths (1-9) dialog.
func ShowQuickPathsDialog(pages *tview.Pages, paths map[string]string, cb QuickPathsCallbacks) {
	table := tview.NewTable()
	table.SetBackgroundColor(theme.ColorDialogBg)
	table.SetSelectable(true, false)
	table.SetSelectedStyle(tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.NewRGBColor(0, 170, 170)))

	for i := 1; i <= 9; i++ {
		key := fmt.Sprintf("%d", i)
		path := "\u2014" // em dash
		if p, ok := paths[key]; ok && p != "" {
			path = p
		}
		slotCell := tview.NewTableCell(fmt.Sprintf(" %d ", i)).
			SetTextColor(tcell.ColorYellow).
			SetBackgroundColor(theme.ColorDialogBg)
		pathCell := tview.NewTableCell(path).
			SetTextColor(theme.ColorDialogFg).
			SetBackgroundColor(theme.ColorDialogBg).
			SetExpansion(1)
		table.SetCell(i-1, 0, slotCell)
		table.SetCell(i-1, 1, pathCell)
	}

	table.SetSelectedFunc(func(row, col int) {
		slot := row + 1
		if slot >= 1 && slot <= 9 {
			cb.OnEdit(slot, paths[fmt.Sprintf("%d", slot)])
		}
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			cb.OnClose()
			return nil
		case tcell.KeyRune:
			row, _ := table.GetSelection()
			slot := row + 1
			switch event.Rune() {
			case 's', 'S':
				if slot >= 1 && slot <= 9 {
					cb.OnSet(slot)
				}
				return nil
			case 'e', 'E':
				if slot >= 1 && slot <= 9 {
					cb.OnEdit(slot, paths[fmt.Sprintf("%d", slot)])
				}
				return nil
			case 'd', 'D':
				if slot >= 1 && slot <= 9 {
					cb.OnDelete(slot)
				}
				return nil
			case 'g', 'G':
				if slot >= 1 && slot <= 9 {
					cb.OnGo(slot)
				}
				return nil
			case '1', '2', '3', '4', '5', '6', '7', '8', '9':
				n := int(event.Rune() - '0')
				cb.OnGo(n)
				return nil
			}
		}
		return event
	})

	frame := tview.NewFrame(table).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBorderColor(theme.ColorDialogBorder)
	frame.SetBackgroundColor(theme.ColorDialogBg)
	frame.SetTitle(" Quick Paths ")
	frame.SetTitleColor(theme.ColorHeaderFg)
	helpText := " S-Set  E-Edit  D-Delete  G-Go  Esc-Close "
	frame.AddText(helpText, false, tview.AlignCenter, tcell.ColorYellow)

	dialogWidth := 60
	dialogHeight := 9 + 4 // 9 rows + border + help

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(frame, dialogWidth, 0, true).
			AddItem(nil, 0, 1, false),
			dialogHeight, 0, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("quickpaths", flex, true, true)
}
