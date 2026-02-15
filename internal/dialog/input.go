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

	// Calculate dialog width based on content
	dialogWidth := len(label) + len(defaultValue) + 10
	if dialogWidth < 60 {
		dialogWidth = 60
	}
	if dialogWidth > 100 {
		dialogWidth = 100
	}
	fieldWidth := dialogWidth - len(label) - 6

	form.AddInputField(label, defaultValue, fieldWidth, nil, nil)
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
			AddItem(form, dialogWidth, 0, true).
			AddItem(nil, 0, 1, false),
			7, 0, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("input", flex, true, true)
}
