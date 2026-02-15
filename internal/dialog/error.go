package dialog

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// ShowError displays an error message dialog.
func ShowError(pages *tview.Pages, message string, onClose func()) {
	modal := tview.NewModal().
		SetText(message).
		AddButtons([]string{"OK"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			if onClose != nil {
				onClose()
			}
		})

	modal.SetBackgroundColor(theme.ColorDialogBg)
	modal.SetTextColor(theme.ColorDialogFg)
	modal.SetButtonBackgroundColor(theme.ColorButtonBg)
	modal.SetButtonTextColor(theme.ColorButtonFg)
	modal.SetBorderColor(theme.ColorDialogBorder)
	modal.SetTitle(" Error ")
	modal.SetBorder(true)
	modal.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape || event.Key() == tcell.KeyEnter {
			if onClose != nil {
				onClose()
			}
			return nil
		}
		return event
	})

	pages.AddPage("error", modal, true, true)
}
