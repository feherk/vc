package dialog

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// ShowConfirm displays a Yes/No confirmation dialog.
func ShowConfirm(pages *tview.Pages, title, message string, callback func(bool)) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"Yes", "No"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			callback(buttonLabel == "Yes")
		})

	modal.SetBackgroundColor(theme.ColorDialogBg)
	modal.SetTextColor(theme.ColorDialogFg)
	modal.SetButtonBackgroundColor(theme.ColorButtonBg)
	modal.SetButtonTextColor(theme.ColorButtonFg)
	modal.SetBorderColor(theme.ColorDialogBorder)
	modal.SetTitle(" " + title + " ")
	modal.SetBorder(true)
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			callback(false)
			return nil
		}
		return event
	})

	pages.AddPage("confirm", modal, true, true)
}
