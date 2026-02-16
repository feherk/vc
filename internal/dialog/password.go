package dialog

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// ShowPasswordDialog displays a password input dialog.
// If confirm is true, it shows two fields and validates they match.
func ShowPasswordDialog(pages *tview.Pages, title string, confirm bool, callback func(password string), onCancel func()) {
	form := tview.NewForm()
	form.SetBackgroundColor(theme.ColorDialogBg)
	form.SetFieldBackgroundColor(theme.ColorPanelBg)
	form.SetFieldTextColor(tcell.ColorWhite)
	form.SetLabelColor(theme.ColorDialogFg)
	form.SetButtonBackgroundColor(theme.ColorButtonBg)
	form.SetButtonTextColor(theme.ColorButtonFg)
	form.SetBorderColor(theme.ColorDialogBorder)
	form.SetTitle(" " + title + " ")
	form.SetTitleColor(theme.ColorHeaderFg)
	form.SetBorder(true)

	form.AddPasswordField("Password:", "", 30, '*', nil)
	if confirm {
		form.AddPasswordField("Confirm:", "", 30, '*', nil)
	}

	errText := tview.NewTextView()
	errText.SetBackgroundColor(theme.ColorDialogBg)
	errText.SetTextColor(tcell.ColorRed)
	errText.SetTextAlign(tview.AlignCenter)

	submit := func() {
		pw := form.GetFormItem(0).(*tview.InputField).GetText()
		if pw == "" {
			errText.SetText("Password cannot be empty")
			return
		}
		if confirm {
			pw2 := form.GetFormItem(1).(*tview.InputField).GetText()
			if pw != pw2 {
				errText.SetText("Passwords do not match")
				return
			}
		}
		callback(pw)
	}

	form.AddButton("OK", submit)
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

	dialogWidth := 50
	dialogHeight := 7
	if confirm {
		dialogHeight = 9
	}

	inner := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(form, dialogHeight-1, 0, true).
		AddItem(errText, 1, 0, false)

	frame := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(inner, dialogWidth, 0, true).
			AddItem(nil, 0, 1, false),
			dialogHeight, 0, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("password", frame, true, true)
}
