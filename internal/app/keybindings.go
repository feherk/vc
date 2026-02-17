package app

import (
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/panel"
)

// SetupKeyBindings configures global key handling for the application.
func (a *App) SetupKeyBindings() {
	a.TviewApp.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		// Modal dialog open — let it handle all keys
		if a.ModalOpen {
			return event
		}

		// Menu is active — let the focused List handle keys
		// (Up/Down/Enter handled by List, Left/Right/Esc by List's InputCapture)
		if a.MenuActive {
			return event
		}

		switch event.Key() {
		case tcell.KeyF1:
			a.ShowServerDialog()
			return nil

		case tcell.KeyF10:
			a.SaveConfig()
			a.TviewApp.Stop()
			return nil

		case tcell.KeyTab:
			a.switchPanel()
			a.CmdLine.SetPath(a.GetActivePanel().Path)
			return nil

		case tcell.KeyF2:
			a.CompressFiles()
			return nil

		case tcell.KeyF3:
			a.ViewFile()
			return nil

		case tcell.KeyF4:
			a.EditFile()
			return nil

		case tcell.KeyF5:
			a.CopyFiles()
			return nil

		case tcell.KeyF6:
			a.MoveFiles()
			return nil

		case tcell.KeyF7:
			a.MakeDir()
			return nil

		case tcell.KeyF8, tcell.KeyDelete:
			a.DeleteFiles()
			return nil

		case tcell.KeyF9:
			a.ActivateMenu()
			return nil

		case tcell.KeyF12:
			e := a.GetActivePanel().CurrentEntry()
			if e != nil && e.Name != ".." {
				if !a.CmdLineFocused {
					a.CmdLineFocused = true
					a.CmdLine.SetText("")
					a.TviewApp.SetFocus(a.CmdLine)
				}
				a.CmdLine.SetText(a.CmdLine.GetText() + e.Name)
			}
			return nil

		case tcell.KeyCtrlR:
			a.GetActivePanel().Refresh()
			a.GetInactivePanel().Refresh()
			return nil

		case tcell.KeyInsert, tcell.KeyF11:
			a.GetActivePanel().ToggleSelection()
			return nil

		case tcell.KeyBackspace, tcell.KeyBackspace2:
			if a.CmdLineFocused {
				if a.CmdLine.GetText() == "" {
					a.CmdLineFocused = false
					a.focusActiveTable()
					return nil
				}
				return event
			}
			p := a.GetActivePanel()
			if atRoot := p.GoParent(); atRoot && !p.IsRemote() {
				a.ShowDriveSelector()
			}
			a.CmdLine.SetPath(p.Path)
			return nil

		case tcell.KeyEnter:
			if a.CmdLineFocused {
				return event
			}
			p := a.GetActivePanel()
			entry, atRoot := p.Enter()
			if atRoot && !p.IsRemote() {
				a.ShowDriveSelector()
			} else if entry != nil {
				a.OpenFile()
				go func() {
					time.Sleep(2 * time.Second)
					a.TviewApp.QueueUpdateDraw(func() {
						a.GetActivePanel().Refresh()
						a.GetInactivePanel().Refresh()
					})
				}()
			}
			a.CmdLine.SetPath(p.Path)
			return nil

		case tcell.KeyCtrlS:
			if !a.CmdLineFocused {
				a.QuickSearch()
				return nil
			}

		case tcell.KeyCtrlN:
			if !a.CmdLineFocused {
				a.ShowQuickPathsDialog()
				return nil
			}

		case tcell.KeyRune:
			if event.Rune() == ' ' && !a.CmdLineFocused {
				a.CalcDirSize()
				return nil
			}
			if !a.CmdLineFocused {
				a.FocusCmdLine(event.Rune())
				return nil
			}
		}

		return event
	})

	// App-level mouse capture: dismiss menu on outside click, block clicks during modals
	a.TviewApp.SetMouseCapture(func(event *tcell.EventMouse, action tview.MouseAction) (*tcell.EventMouse, tview.MouseAction) {
		if action != tview.MouseLeftClick {
			return event, action
		}
		if a.MenuActive {
			mx, my := event.Position()
			// Click on menu bar → let MenuBar.OnClick handle (switches menu)
			if my == 0 {
				return event, action
			}
			// Click inside dropdown → let Dropdown.MouseHandler handle
			if a.activeDropdown != nil {
				dx, dy, dw, dh := a.activeDropdown.GetRect()
				if mx >= dx && mx < dx+dw && my >= dy && my < dy+dh {
					return event, action
				}
			}
			// Click outside both → dismiss menu
			a.DeactivateMenu()
			go a.TviewApp.QueueUpdateDraw(func() {})
			return nil, action
		}
		return event, action
	})

	// Panel table selection change handler
	a.LeftPanel.Table.SetSelectionChangedFunc(func(row, col int) {
		a.LeftPanel.HandleSelectionChanged(row, col)
	})
	a.RightPanel.Table.SetSelectionChangedFunc(func(row, col int) {
		a.RightPanel.HandleSelectionChanged(row, col)
	})

	// Double-click = Enter. Two detection methods:
	// 1. tview's MouseLeftDoubleClick (500ms threshold)
	// 2. Custom detection on MouseLeftClick as backup (800ms threshold)
	doMouseEnter := func(p *panel.Panel) {
		entry, atRoot := p.Enter()
		if atRoot && !p.IsRemote() {
			a.ShowDriveSelector()
		} else if entry != nil {
			a.OpenFile()
			go func() {
				time.Sleep(2 * time.Second)
				a.TviewApp.QueueUpdateDraw(func() {
					a.GetActivePanel().Refresh()
					a.GetInactivePanel().Refresh()
				})
			}()
		}
		a.CmdLine.SetPath(p.Path)
		// Force redraw — Table doesn't consume MouseLeftDoubleClick
		// so tview skips the draw call after fireMouseActions.
		go a.TviewApp.QueueUpdateDraw(func() {})
	}

	setupPanelMouse := func(p *panel.Panel) {
		p.Table.SetMouseCapture(func(action tview.MouseAction, event *tcell.EventMouse) (tview.MouseAction, *tcell.EventMouse) {
			if a.ModalOpen || a.MenuActive {
				return action, event
			}

			switch action {
			case tview.MouseLeftDoubleClick:
				a.lastClickTime = time.Time{}
				doMouseEnter(p)
				return action, nil

			case tview.MouseRightClick:
				_, my := event.Position()
				_, tableY, _, _ := p.Table.GetInnerRect()
				row := my - tableY
				if p.Mode != panel.ModeBrief {
					row--
				}
				p.ToggleSelectionAt(row)
				go a.TviewApp.QueueUpdateDraw(func() {})
				return action, nil

			case tview.MouseLeftClick:
				_, my := event.Position()
				_, tableY, _, _ := p.Table.GetInnerRect()
				row := my - tableY
				if p.Mode != panel.ModeBrief {
					row--
				}
				now := time.Now()
				if now.Sub(a.lastClickTime) < 800*time.Millisecond &&
					a.lastClickRow == row && a.lastClickTable == p.Table {
					a.lastClickTime = time.Time{}
					doMouseEnter(p)
					return action, nil
				}
				a.lastClickTime = now
				a.lastClickRow = row
				a.lastClickTable = p.Table
				return action, event
			}

			return action, event
		})
	}
	setupPanelMouse(a.LeftPanel)
	setupPanelMouse(a.RightPanel)

	// Track active panel on focus (ignore during menu/modal to prevent spurious switches)
	a.LeftPanel.Table.SetFocusFunc(func() {
		if !a.MenuActive && !a.ModalOpen && a.activePanel != 0 {
			a.activePanel = 0
			a.updatePanelStates()
		}
	})
	a.RightPanel.Table.SetFocusFunc(func() {
		if !a.MenuActive && !a.ModalOpen && a.activePanel != 1 {
			a.activePanel = 1
			a.updatePanelStates()
		}
	})
}

// switchPanel toggles the active panel.
func (a *App) switchPanel() {
	a.CmdLineFocused = false
	if a.activePanel == 0 {
		a.activePanel = 1
		a.TviewApp.SetFocus(a.RightPanel.Table)
	} else {
		a.activePanel = 0
		a.TviewApp.SetFocus(a.LeftPanel.Table)
	}
	a.updatePanelStates()
}

func (a *App) updatePanelStates() {
	a.LeftPanel.SetActive(a.activePanel == 0)
	a.RightPanel.SetActive(a.activePanel == 1)
}
