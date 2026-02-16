package dialog

import (
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
	table := tview.NewTable()
	table.SetBackgroundColor(theme.ColorDialogBg)
	table.SetSelectable(true, false)
	table.SetSelectedStyle(tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.NewRGBColor(0, 170, 170)))

	for i, srv := range servers {
		prefix := "  "
		if cb.IsConnected != nil && cb.IsConnected(srv.Name) {
			prefix = "* "
		}
		cell := tview.NewTableCell(prefix + srv.Name).
			SetTextColor(theme.ColorDialogFg).
			SetBackgroundColor(theme.ColorDialogBg).
			SetExpansion(1)
		table.SetCell(i, 0, cell)
	}
	if len(servers) == 0 {
		cell := tview.NewTableCell("  (no servers configured)").
			SetTextColor(theme.ColorDialogFg).
			SetBackgroundColor(theme.ColorDialogBg).
			SetExpansion(1)
		table.SetCell(0, 0, cell)
	}

	// Enter on row → connect
	table.SetSelectedFunc(func(row, col int) {
		if row >= 0 && row < len(servers) {
			cb.OnConnect(servers[row])
		}
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			cb.OnClose()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case 'c', 'C':
				row, _ := table.GetSelection()
				if row >= 0 && row < len(servers) {
					cb.OnConnect(servers[row])
				}
				return nil
			case 'a', 'A':
				cb.OnAdd()
				return nil
			case 'e', 'E':
				row, _ := table.GetSelection()
				if row >= 0 && row < len(servers) {
					cb.OnEdit(row, servers[row])
				}
				return nil
			case 'd', 'D':
				row, _ := table.GetSelection()
				if row >= 0 && row < len(servers) {
					cb.OnDelete(row)
				}
				return nil
			case 'x', 'X':
				row, _ := table.GetSelection()
				if row >= 0 && row < len(servers) {
					cb.OnDisconnect(servers[row].Name)
				}
				return nil
			}
		}
		return event
	})

	frame := tview.NewFrame(table).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBorderColor(theme.ColorDialogBorder)
	frame.SetBackgroundColor(theme.ColorDialogBg)
	frame.SetTitle(" Servers ")
	frame.SetTitleColor(theme.ColorHeaderFg)
	helpText := " C-Connect  A-Add  E-Edit  D-Delete  X-Disconnect  Esc-Close "
	frame.AddText(helpText, false, tview.AlignCenter, tcell.ColorYellow)

	dialogWidth := len(helpText) + 4
	for _, srv := range servers {
		if w := len(srv.Name) + 6; w > dialogWidth {
			dialogWidth = w
		}
	}
	dialogHeight := len(servers) + 4
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
