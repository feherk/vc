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
	OnMove       func(fromIdx, toIdx int)
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

	// Moving mode state
	moving := false
	moveOrigin := -1
	var savedOrder []config.ServerConfig

	renderRows := func() {
		table.Clear()
		for i, srv := range servers {
			prefix := "  "
			if cb.IsConnected != nil && cb.IsConnected(srv.Name) {
				prefix = "* "
			}
			if moving && i == moveOrigin {
				prefix = "> "
			}
			fg := theme.ColorDialogFg
			bg := theme.ColorDialogBg
			if moving && i == moveOrigin {
				fg = tcell.ColorYellow
			}
			cell := tview.NewTableCell(prefix + srv.Name).
				SetTextColor(fg).
				SetBackgroundColor(bg).
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
	}
	renderRows()

	helpNormal := " C-Connect  A-Add  E-Edit  D-Delete  M-Move  X-Disc  Esc-Close "
	helpMoving := " \u2191\u2193-Move  M/Enter-Drop  Esc-Cancel "

	frame := tview.NewFrame(table).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBorderColor(theme.ColorDialogBorder)
	frame.SetBackgroundColor(theme.ColorDialogBg)
	frame.SetTitle(" Servers ")
	frame.SetTitleColor(theme.ColorHeaderFg)
	frame.AddText(helpNormal, false, tview.AlignCenter, tcell.ColorYellow)

	updateHelp := func(text string) {
		frame.Clear()
		frame.AddText(text, false, tview.AlignCenter, tcell.ColorYellow)
	}

	// Enter on row → connect (only when not moving)
	table.SetSelectedFunc(func(row, col int) {
		if moving {
			// Drop
			if cb.OnMove != nil {
				cb.OnMove(0, 0)
			}
			savedOrder = nil
			moving = false
			moveOrigin = -1
			renderRows()
			updateHelp(helpNormal)
			return
		}
		if row >= 0 && row < len(servers) {
			cb.OnConnect(servers[row])
		}
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		if moving {
			row, _ := table.GetSelection()
			switch event.Key() {
			case tcell.KeyUp:
				if moveOrigin > 0 {
					servers[moveOrigin], servers[moveOrigin-1] = servers[moveOrigin-1], servers[moveOrigin]
					moveOrigin--
					renderRows()
					table.Select(moveOrigin, 0)
				}
				return nil
			case tcell.KeyDown:
				if moveOrigin < len(servers)-1 {
					servers[moveOrigin], servers[moveOrigin+1] = servers[moveOrigin+1], servers[moveOrigin]
					moveOrigin++
					renderRows()
					table.Select(moveOrigin, 0)
				}
				return nil
			case tcell.KeyEscape:
				// Cancel move — restore original order
				copy(servers, savedOrder)
				savedOrder = nil
				moving = false
				moveOrigin = -1
				renderRows()
				updateHelp(helpNormal)
				return nil
			case tcell.KeyRune:
				if event.Rune() == 'm' || event.Rune() == 'M' {
					// Drop at current position
					if cb.OnMove != nil {
						cb.OnMove(row, moveOrigin)
					}
					moving = false
					moveOrigin = -1
					renderRows()
					updateHelp(helpNormal)
					return nil
				}
			}
			return nil // Eat all other keys during move
		}

		// Normal mode
		switch event.Key() {
		case tcell.KeyEscape:
			cb.OnClose()
			return nil
		case tcell.KeyRune:
			row, _ := table.GetSelection()
			switch event.Rune() {
			case 'c', 'C':
				if row >= 0 && row < len(servers) {
					cb.OnConnect(servers[row])
				}
				return nil
			case 'a', 'A':
				cb.OnAdd()
				return nil
			case 'e', 'E':
				if row >= 0 && row < len(servers) {
					cb.OnEdit(row, servers[row])
				}
				return nil
			case 'd', 'D':
				if row >= 0 && row < len(servers) {
					cb.OnDelete(row)
				}
				return nil
			case 'x', 'X':
				if row >= 0 && row < len(servers) {
					cb.OnDisconnect(servers[row].Name)
				}
				return nil
			case 'm', 'M':
				if row >= 0 && row < len(servers) {
					moving = true
					moveOrigin = row
					savedOrder = make([]config.ServerConfig, len(servers))
					copy(savedOrder, servers)
					renderRows()
					updateHelp(helpMoving)
				}
				return nil
			}
		}
		return event
	})

	dialogWidth := len(helpNormal) + 4
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
