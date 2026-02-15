package dialog

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// ShowInput displays a text input dialog.
func ShowInput(pages *tview.Pages, title, label, defaultValue string, onOK func(string), onCancel func()) {
	form := tview.NewForm()
	form.SetBackgroundColor(theme.ColorDialogBg)
	form.SetFieldBackgroundColor(theme.ColorPanelBg)
	form.SetFieldTextColor(tcell.ColorWhite)
	form.SetLabelColor(theme.ColorDialogFg)
	form.SetButtonBackgroundColor(theme.ColorButtonBg)
	form.SetButtonTextColor(theme.ColorButtonFg)
	form.SetBorderColor(theme.ColorDialogBorder)
	form.SetTitle(" " + title + " ")
	form.SetBorder(true)

	form.AddInputField(label, defaultValue, 50, nil, nil)
	form.AddButton("OK", func() {
		value := form.GetFormItem(0).(*tview.InputField).GetText()
		onOK(value)
	})
	form.AddButton("Cancel", func() {
		onCancel()
	})

	form.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyEscape {
			onCancel()
			return nil
		}
		return event
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, 60, 0, true).
			AddItem(nil, 0, 1, false),
			7, 0, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("input", flex, true, true)
}
