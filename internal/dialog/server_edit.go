package dialog

import (
	"strconv"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/config"
	"github.com/feherkaroly/vc/internal/theme"
)

// ShowServerEdit displays the server add/edit form.
func ShowServerEdit(pages *tview.Pages, title string, srv config.ServerConfig, onSave func(config.ServerConfig), onCancel func()) {
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

	portStr := ""
	if srv.Port != 0 {
		portStr = strconv.Itoa(srv.Port)
	}

	// Protocol selection
	protocols := []string{"sftp", "ftp", "ftps"}
	initialProtocol := 0
	for i, p := range protocols {
		if p == srv.Protocol {
			initialProtocol = i
			break
		}
	}

	form.AddInputField("Name:", srv.Name, 30, nil, nil)
	form.AddDropDown("Protocol:", protocols, initialProtocol, nil)
	form.AddInputField("Host:", srv.Host, 30, nil, nil)
	form.AddInputField("Port:", portStr, 10, nil, nil)
	form.AddInputField("User:", srv.User, 30, nil, nil)
	form.AddPasswordField("Password:", srv.Password, 30, '*', nil)
	form.AddInputField("Key Path:", srv.KeyPath, 40, nil, nil)

	form.AddButton("Save", func() {
		name := form.GetFormItem(0).(*tview.InputField).GetText()
		_, protocol := form.GetFormItem(1).(*tview.DropDown).GetCurrentOption()
		host := form.GetFormItem(2).(*tview.InputField).GetText()
		portText := form.GetFormItem(3).(*tview.InputField).GetText()
		user := form.GetFormItem(4).(*tview.InputField).GetText()
		password := form.GetFormItem(5).(*tview.InputField).GetText()
		keyPath := form.GetFormItem(6).(*tview.InputField).GetText()

		port := 0
		if portText != "" {
			port, _ = strconv.Atoi(portText)
		}

		onSave(config.ServerConfig{
			Name:     name,
			Protocol: protocol,
			Host:     host,
			Port:     port,
			User:     user,
			Password: password,
			KeyPath:  keyPath,
		})
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

	dialogWidth := 60
	dialogHeight := 19

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(form, dialogWidth, 0, true).
			AddItem(nil, 0, 1, false),
			dialogHeight, 0, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("server_edit", flex, true, true)
}
