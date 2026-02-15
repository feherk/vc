package dialog

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// ShowDriveDialog displays a list of available drives for the user to select.
func ShowDriveDialog(pages *tview.Pages, drives []string, onSelect func(string), onCancel func()) {
	list := tview.NewList()
	list.ShowSecondaryText(false)
	list.SetBackgroundColor(theme.ColorDialogBg)
	list.SetMainTextColor(theme.ColorDialogFg)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.NewRGBColor(0, 170, 170))
	list.SetHighlightFullLine(true)

	for _, d := range drives {
		drive := d
		list.AddItem("  "+drive, "", 0, func() {
			onSelect(drive)
		})
	}

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			onCancel()
			return nil
		}
		return event
	})

	frame := tview.NewFrame(list).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBorderColor(theme.ColorDialogBorder)
	frame.SetBackgroundColor(theme.ColorDialogBg)
	frame.SetTitle(" Change Drive ")
	frame.SetTitleColor(theme.ColorHeaderFg)

	dialogWidth := 24
	dialogHeight := len(drives) + 4
	if dialogHeight > 20 {
		dialogHeight = 20
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(frame, dialogWidth, 0, true).
			AddItem(nil, 0, 1, false),
			dialogHeight, 0, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("drive_dialog", flex, true, true)
}
