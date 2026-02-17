package app

import (
	"time"

	"github.com/gdamore/tcell/v2"
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

	// Panel table selection change handler
	a.LeftPanel.Table.SetSelectionChangedFunc(func(row, col int) {
		a.LeftPanel.HandleSelectionChanged(row, col)
	})
	a.RightPanel.Table.SetSelectionChangedFunc(func(row, col int) {
		a.RightPanel.HandleSelectionChanged(row, col)
	})

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
