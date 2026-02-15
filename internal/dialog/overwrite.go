package dialog

import (
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// OverwriteChoice represents the user's choice for file overwrite.
type OverwriteChoice int

const (
	OverwriteYes OverwriteChoice = iota
	OverwriteNo
	OverwriteAll
	OverwriteCancel
)

// ShowOverwrite displays an overwrite confirmation dialog.
func ShowOverwrite(pages *tview.Pages, filename string, callback func(OverwriteChoice)) {
	modal := tview.NewModal().
		SetText("Overwrite " + filename + "?").
		AddButtons([]string{"Yes", "No", "All", "Cancel"}).
		SetDoneFunc(func(buttonIndex int, buttonLabel string) {
			switch buttonLabel {
			case "Yes":
				callback(OverwriteYes)
			case "No":
				callback(OverwriteNo)
			case "All":
				callback(OverwriteAll)
			default:
				callback(OverwriteCancel)
			}
		})

	modal.SetBackgroundColor(theme.ColorDialogBg)
	modal.SetTextColor(theme.ColorDialogFg)
	modal.SetButtonBackgroundColor(theme.ColorButtonBg)
	modal.SetButtonTextColor(theme.ColorButtonFg)
	modal.SetBorderColor(theme.ColorDialogBorder)
	modal.SetTitle(" Overwrite ")
	modal.SetBorder(true)

	pages.AddPage("overwrite", modal, true, true)
}
