package dialog

import (
	"fmt"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/config"
	"github.com/feherkaroly/vc/internal/theme"
)

// ServerDialogCallbacks holds all callbacks for server dialog actions.
type ServerDialogCallbacks struct {
	OnConnect    func(cfg config.ServerConfig)
	OnDisconnect func(name string)
	OnAdd        func()
	OnEdit       func(idx int, cfg config.ServerConfig)
	OnDelete     func(idx int)
	OnClose      func()
	IsConnected  func(name string) bool
}

// ShowServerDialog displays the server list dialog.
func ShowServerDialog(pages *tview.Pages, servers []config.ServerConfig, cb ServerDialogCallbacks) {
	list := tview.NewList()
	list.ShowSecondaryText(true)
	list.SetBackgroundColor(theme.ColorDialogBg)
	list.SetMainTextColor(theme.ColorDialogFg)
	list.SetSecondaryTextColor(tcell.NewRGBColor(0, 170, 170))
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.NewRGBColor(0, 170, 170))
	list.SetHighlightFullLine(true)

	for _, srv := range servers {
		prefix := "  "
		if cb.IsConnected != nil && cb.IsConnected(srv.Name) {
			prefix = "* "
		}
		label := prefix + srv.Name
		detail := fmt.Sprintf("  %s://%s@%s", srv.Protocol, srv.User, srv.Host)
		if srv.Port != 0 {
			detail += fmt.Sprintf(":%d", srv.Port)
		}
		list.AddItem(label, detail, 0, nil)
	}
	if len(servers) == 0 {
		list.AddItem("  (no servers configured)", "", 0, nil)
	}

	// Enter on list item → connect
	list.SetSelectedFunc(func(idx int, _, _ string, _ rune) {
		if idx >= 0 && idx < len(servers) {
			cb.OnConnect(servers[idx])
		}
	})

	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			cb.OnClose()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'c', 'C':
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(servers) {
					cb.OnConnect(servers[idx])
				}
				return nil
			case 'a', 'A':
				cb.OnAdd()
				return nil
			case 'e', 'E':
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(servers) {
					cb.OnEdit(idx, servers[idx])
				}
				return nil
			case 'd', 'D':
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(servers) {
					cb.OnDelete(idx)
				}
				return nil
			case 'x', 'X':
				idx := list.GetCurrentItem()
				if idx >= 0 && idx < len(servers) {
					cb.OnDisconnect(servers[idx].Name)
				}
				return nil
			}
		}
		return event
	})

	frame := tview.NewFrame(list).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBorderColor(theme.ColorDialogBorder)
	frame.SetBackgroundColor(theme.ColorDialogBg)
	frame.SetTitle(" Servers ")
	frame.SetTitleColor(theme.ColorHeaderFg)
	frame.AddText(" C-Connect  A-Add  E-Edit  D-Delete  X-Disconnect  Esc-Close ", false, tview.AlignCenter, tcell.ColorYellow)

	dialogWidth := 64
	dialogHeight := len(servers)*2 + 6
	if dialogHeight < 10 {
		dialogHeight = 10
	}
	if dialogHeight > 22 {
		dialogHeight = 22
	}

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(frame, dialogWidth, 0, true).
			AddItem(nil, 0, 1, false),
			dialogHeight, 0, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("server_dialog", flex, true, true)
}
