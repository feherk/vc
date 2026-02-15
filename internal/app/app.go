package app

import (
	"archive/zip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/cmdline"
	"github.com/feherkaroly/vc/internal/config"
	"github.com/feherkaroly/vc/internal/dialog"
	"github.com/feherkaroly/vc/internal/fileops"
	"github.com/feherkaroly/vc/internal/fnbar"
	"github.com/feherkaroly/vc/internal/menu"
	"github.com/feherkaroly/vc/internal/model"
	"github.com/feherkaroly/vc/internal/panel"
	"github.com/feherkaroly/vc/internal/theme"
	"github.com/feherkaroly/vc/internal/vfs"
	"github.com/feherkaroly/vc/internal/viewer"
)

// App is the main application struct that ties everything together.
type App struct {
	TviewApp *tview.Application
	Pages    *tview.Pages

	LeftPanel  *panel.Panel
	RightPanel *panel.Panel

	MenuBar *menu.MenuBar
	FnBar   *fnbar.FnBar
	CmdLine *cmdline.CmdLine
	ConnMgr *vfs.ConnMgr

	activePanel    int // 0 = left, 1 = right
	MenuActive     bool
	ModalOpen      bool
	CmdLineFocused bool
}

var Version string

// New creates and initializes the application.
func New(leftPath, rightPath string) *App {
	a := &App{
		TviewApp: tview.NewApplication(),
		ConnMgr:  vfs.NewConnMgr(),
	}

	a.LeftPanel = panel.NewPanel(leftPath, vfs.NewLocalFS())
	a.RightPanel = panel.NewPanel(rightPath, vfs.NewLocalFS())
	a.MenuBar = menu.NewMenuBar()
	a.MenuBar.Version = Version
	a.FnBar = fnbar.New()

	cfg := config.Load()
	a.LeftPanel.Mode = panel.DisplayMode(cfg.LeftPanel.Mode)
	a.LeftPanel.SortMode = panel.SortMode(cfg.LeftPanel.SortMode)
	a.RightPanel.Mode = panel.DisplayMode(cfg.RightPanel.Mode)
	a.RightPanel.SortMode = panel.SortMode(cfg.RightPanel.SortMode)
	a.LeftPanel.Refresh()
	a.RightPanel.Refresh()

	a.CmdLine = cmdline.New()
	a.CmdLine.SetPath(a.LeftPanel.Path)
	a.CmdLine.SetExecuteFunc(func(cmd string) {
		a.ExecuteCommand(cmd)
	})
	a.CmdLine.SetFocusFunc(func(focused bool) {
		a.CmdLineFocused = focused
		if !focused {
			a.TviewApp.SetFocus(a.activeTable())
		}
	})

	a.buildLayout()

	a.LeftPanel.SetActive(true)
	a.RightPanel.SetActive(false)
	a.activePanel = 0

	a.SetupKeyBindings()
	a.TviewApp.SetFocus(a.LeftPanel.Table)

	return a
}

func (a *App) buildLayout() {
	panelFlex := tview.NewFlex().SetDirection(tview.FlexColumn).
		AddItem(a.LeftPanel.Box, 0, 1, true).
		AddItem(a.RightPanel.Box, 0, 1, false)

	mainFlex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(a.MenuBar, 1, 0, false).
		AddItem(panelFlex, 0, 1, true).
		AddItem(a.CmdLine, 1, 0, false).
		AddItem(a.FnBar, 1, 0, false)

	a.Pages = tview.NewPages().
		AddPage("main", mainFlex, true, true)
}

// Run starts the application.
func (a *App) Run() error {
	defer a.ConnMgr.DisconnectAll()
	return a.TviewApp.SetRoot(a.Pages, true).Run()
}

// GetActivePanel returns the currently focused panel.
func (a *App) GetActivePanel() *panel.Panel {
	if a.activePanel == 0 {
		return a.LeftPanel
	}
	return a.RightPanel
}

// GetInactivePanel returns the non-focused panel.
func (a *App) GetInactivePanel() *panel.Panel {
	if a.activePanel == 0 {
		return a.RightPanel
	}
	return a.LeftPanel
}

func (a *App) activeTable() *tview.Table {
	return a.GetActivePanel().Table
}

func (a *App) focusActiveTable() {
	a.TviewApp.SetFocus(a.activeTable())
}

// showDialog adds a dialog page and focuses it.
func (a *App) showDialog(name string, p tview.Primitive) {
	a.ModalOpen = true
	a.Pages.AddPage(name, p, true, true)
	a.TviewApp.SetFocus(p)
}

// closeDialog removes a dialog page and restores focus to the correct panel.
func (a *App) closeDialog(name string) {
	saved := a.activePanel
	a.Pages.RemovePage(name)
	a.ModalOpen = false
	a.activePanel = saved
	a.focusActiveTable()
	a.updatePanelStates()
}

// ViewFile opens the F3 file viewer. For .zip files it shows the archive contents.
func (a *App) ViewFile() {
	p := a.GetActivePanel()
	e := p.CurrentEntry()
	if e == nil || e.IsDir {
		return
	}

	path := p.FS.Join(p.Path, e.Name)

	if strings.HasSuffix(strings.ToLower(e.Name), ".zip") {
		if p.IsRemote() {
			a.showRemoteError("View zip")
			return
		}
		a.viewZipContents(path)
		return
	}

	if p.IsRemote() {
		// Remote file: read via VFS and display as text
		data, err := p.FS.ReadFile(path)
		if err != nil {
			dialog.ShowError(a.Pages, "Read error: "+err.Error(), func() {
				a.closeDialog("error")
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
			return
		}
		v := viewer.NewFromText(path, string(data))
		v.SetDoneFunc(func() {
			a.closeDialog("viewer")
		})
		a.showDialog("viewer", v)
		return
	}

	v := viewer.New(path)

	v.SetDoneFunc(func() {
		a.closeDialog("viewer")
	})

	a.showDialog("viewer", v)
}

// viewZipContents shows the file listing of a zip archive.
func (a *App) viewZipContents(path string) {
	r, err := zip.OpenReader(path)
	if err != nil {
		dialog.ShowError(a.Pages, "Cannot open zip: "+err.Error(), func() {
			a.closeDialog("error")
		})
		a.ModalOpen = true
		a.TviewApp.SetFocus(a.Pages)
		return
	}
	defer r.Close()

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("Archive: %s\n", filepath.Base(path)))
	buf.WriteString(fmt.Sprintf("%-12s %-20s %s\n", "Size", "Modified", "Name"))
	buf.WriteString(strings.Repeat("─", 70) + "\n")

	var totalSize uint64
	for _, f := range r.File {
		totalSize += f.UncompressedSize64
		buf.WriteString(fmt.Sprintf("%-12d %-20s %s\n",
			f.UncompressedSize64,
			f.Modified.Format("2006-01-02 15:04:05"),
			f.Name))
	}
	buf.WriteString(strings.Repeat("─", 70) + "\n")
	buf.WriteString(fmt.Sprintf("%d file(s), %d bytes total\n", len(r.File), totalSize))

	v := viewer.NewFromText(path, buf.String())
	v.SetDoneFunc(func() {
		a.closeDialog("viewer")
	})
	a.showDialog("viewer", v)
}

// ZipFiles handles F2 — compress selected items or current item into a zip archive.
func (a *App) ZipFiles() {
	p := a.GetActivePanel()
	if p.IsRemote() {
		a.showRemoteError("Zip")
		return
	}
	entries := p.GetSelectedOrCurrent()
	if len(entries) == 0 {
		return
	}

	var zipName string
	if p.Selection.Count() > 0 {
		zipName = fmt.Sprintf("archiv_%d.zip", time.Now().UnixNano())
	} else {
		name := entries[0].Name
		ext := filepath.Ext(name)
		if ext != "" {
			name = strings.TrimSuffix(name, ext)
		}
		zipName = name + ".zip"
	}
	zipPath := filepath.Join(p.Path, zipName)

	go func() {
		err := createZip(zipPath, p.Path, entries)
		a.TviewApp.QueueUpdateDraw(func() {
			if err != nil {
				dialog.ShowError(a.Pages, "Zip error: "+err.Error(), func() {
					a.closeDialog("error")
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
				return
			}
			p.Selection.Clear()
			p.NavigateTo(p.Path, zipName)
			a.GetInactivePanel().Refresh()
		})
	}()
}

// createZip creates a zip archive at zipPath containing the given entries from baseDir.
func createZip(zipPath, baseDir string, entries []model.FileEntry) error {
	f, err := os.Create(zipPath)
	if err != nil {
		return err
	}
	defer f.Close()

	w := zip.NewWriter(f)
	defer w.Close()

	for _, entry := range entries {
		srcPath := filepath.Join(baseDir, entry.Name)
		if entry.IsDir {
			err = addDirToZip(w, srcPath, entry.Name)
		} else {
			err = addFileToZip(w, srcPath, entry.Name)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func addFileToZip(w *zip.Writer, filePath, nameInZip string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	header, err := zip.FileInfoHeader(info)
	if err != nil {
		return err
	}
	header.Name = nameInZip
	header.Method = zip.Deflate

	writer, err := w.CreateHeader(header)
	if err != nil {
		return err
	}

	_, err = io.Copy(writer, f)
	return err
}

func addDirToZip(w *zip.Writer, dirPath, prefix string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip symlinks
		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		rel, err := filepath.Rel(filepath.Dir(dirPath), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			_, err = w.Create(rel + "/")
			return err
		}

		// Skip non-regular files
		if !info.Mode().IsRegular() {
			return nil
		}

		return addFileToZip(w, path, rel)
	})
}

// OpenFile opens the current file with the system default application.
func (a *App) OpenFile() {
	p := a.GetActivePanel()
	if p.IsRemote() {
		a.showRemoteError("Open")
		return
	}
	e := p.CurrentEntry()
	if e == nil || e.IsDir {
		return
	}
	path := filepath.Join(p.Path, e.Name)
	exec.Command("open", path).Start()
}

// EditFile opens the file in $EDITOR using Suspend.
func (a *App) EditFile() {
	p := a.GetActivePanel()
	if p.IsRemote() {
		a.showRemoteError("Edit")
		return
	}
	e := p.CurrentEntry()
	if e == nil || e.IsDir {
		return
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	path := filepath.Join(p.Path, e.Name)

	a.TviewApp.Suspend(func() {
		cmd := exec.Command(editor, path)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Run()
	})

	p.Refresh()
}

// CopyFiles handles F5.
func (a *App) CopyFiles() {
	src := a.GetActivePanel()
	dst := a.GetInactivePanel()
	entries := src.GetSelectedOrCurrent()
	if len(entries) == 0 {
		return
	}

	desc := entryNames(entries)
	srcFS := src.FS
	dstFS := dst.FS

	dialog.ShowInput(a.Pages, "Copy", "Copy "+desc+" to:", dst.Path, func(target string) {
		a.closeDialog("input")
		if target == "" {
			return
		}

		// Relative path → resolve from destination panel
		if !filepath.IsAbs(target) && dstFS.IsLocal() {
			target = filepath.Join(src.Path, target)
		}

		ctx, cancel := context.WithCancel(context.Background())

		pd := dialog.NewProgressDialog("Copying", func() {
			cancel()
		})
		a.showDialog("progress", pd)

		go func() {
			var lastUpdate time.Time
			fileCount := len(entries)

			for i, entry := range entries {
				srcPath := srcFS.Join(src.Path, entry.Name)
				dstPath := dstFS.Join(target, entry.Name)

				fileIdx := i + 1
				err := fileops.Copy(ctx, srcFS, srcPath, dstFS, dstPath, func(p fileops.Progress) {
					p.FileIndex = fileIdx
					p.FileCount = fileCount
					now := time.Now()
					if now.Sub(lastUpdate) < 100*time.Millisecond {
						return
					}
					lastUpdate = now
					a.TviewApp.QueueUpdateDraw(func() {
						pd.Update(p)
					})
				})
				if err != nil {
					a.TviewApp.QueueUpdateDraw(func() {
						a.closeDialog("progress")
						if ctx.Err() == nil {
							dialog.ShowError(a.Pages, "Copy error: "+err.Error(), func() {
								a.closeDialog("error")
							})
							a.ModalOpen = true
							a.TviewApp.SetFocus(a.Pages)
						}
					})
					return
				}
			}

			a.TviewApp.QueueUpdateDraw(func() {
				a.closeDialog("progress")
				src.Selection.Clear()
				src.Refresh()
				dst.Refresh()
			})
		}()
	}, func() {
		a.closeDialog("input")
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
}

// MoveFiles handles F6.
func (a *App) MoveFiles() {
	src := a.GetActivePanel()
	dst := a.GetInactivePanel()
	entries := src.GetSelectedOrCurrent()
	if len(entries) == 0 {
		return
	}

	desc := entryNames(entries)
	srcFS := src.FS
	dstFS := dst.FS

	defaultTarget := dst.Path
	if len(entries) == 1 {
		defaultTarget = dstFS.Join(dst.Path, entries[0].Name)
	}

	dialog.ShowInput(a.Pages, "Move/Rename", "Move "+desc+" to:", defaultTarget, func(target string) {
		a.closeDialog("input")
		if target == "" {
			return
		}

		// Relative path → resolve from source panel directory
		if !filepath.IsAbs(target) && dstFS.IsLocal() {
			target = filepath.Join(src.Path, target)
		}

		ctx, cancel := context.WithCancel(context.Background())

		pd := dialog.NewProgressDialog("Moving", func() {
			cancel()
		})
		a.showDialog("progress", pd)

		go func() {
			var lastUpdate time.Time
			fileCount := len(entries)

			for i, entry := range entries {
				srcPath := srcFS.Join(src.Path, entry.Name)
				dstPath := target
				if len(entries) > 1 {
					dstPath = dstFS.Join(target, entry.Name)
				}

				fileIdx := i + 1
				err := fileops.Move(ctx, srcFS, srcPath, dstFS, dstPath, func(p fileops.Progress) {
					p.FileIndex = fileIdx
					p.FileCount = fileCount
					now := time.Now()
					if now.Sub(lastUpdate) < 100*time.Millisecond {
						return
					}
					lastUpdate = now
					a.TviewApp.QueueUpdateDraw(func() {
						pd.Update(p)
					})
				})
				if err != nil {
					a.TviewApp.QueueUpdateDraw(func() {
						a.closeDialog("progress")
						if ctx.Err() == nil {
							dialog.ShowError(a.Pages, "Move error: "+err.Error(), func() {
								a.closeDialog("error")
							})
							a.ModalOpen = true
							a.TviewApp.SetFocus(a.Pages)
						}
					})
					return
				}
			}

			a.TviewApp.QueueUpdateDraw(func() {
				a.closeDialog("progress")
				src.Selection.Clear()
				src.Refresh()
				dst.Refresh()
			})
		}()
	}, func() {
		a.closeDialog("input")
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
}

// MakeDir handles F7.
func (a *App) MakeDir() {
	p := a.GetActivePanel()

	dialog.ShowInput(a.Pages, "Make Directory", "Create directory:", "", func(name string) {
		a.closeDialog("input")
		if name == "" {
			return
		}

		err := fileops.MkDir(p.FS, p.FS.Join(p.Path, name))
		if err != nil {
			dialog.ShowError(a.Pages, "MkDir error: "+err.Error(), func() {
				a.closeDialog("error")
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
			return
		}

		p.Refresh()
	}, func() {
		a.closeDialog("input")
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
}

// DeleteFiles handles F8.
func (a *App) DeleteFiles() {
	p := a.GetActivePanel()
	entries := p.GetSelectedOrCurrent()
	if len(entries) == 0 {
		return
	}

	desc := entryNames(entries)

	dialog.ShowConfirm(a.Pages, "Delete", "Delete "+desc+"?", func(yes bool) {
		a.closeDialog("confirm")
		if !yes {
			return
		}

		go func() {
			for _, entry := range entries {
				path := p.FS.Join(p.Path, entry.Name)
				err := fileops.Delete(p.FS, path)
				if err != nil {
					a.TviewApp.QueueUpdateDraw(func() {
						dialog.ShowError(a.Pages, "Delete error: "+err.Error(), func() {
							a.closeDialog("error")
						})
						a.ModalOpen = true
						a.TviewApp.SetFocus(a.Pages)
					})
					return
				}
			}

			a.TviewApp.QueueUpdateDraw(func() {
				p.Selection.Clear()
				p.Refresh()
				a.GetInactivePanel().Refresh()
			})
		}()
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
}

// CalcDirSize calculates directory size with Space key.
func (a *App) CalcDirSize() {
	p := a.GetActivePanel()
	e := p.CurrentEntry()
	if e == nil || !e.IsDir || e.Name == ".." {
		return
	}

	dirPath := p.FS.Join(p.Path, e.Name)

	go func() {
		size := fileops.CalcDirSize(p.FS, dirPath)
		a.TviewApp.QueueUpdateDraw(func() {
			e.DirSize = size
			p.Render()
			p.UpdateTitle()
			p.MoveCursor(1)
		})
	}()
}

// QuickSearch opens a live search input — the cursor moves as you type.
func (a *App) QuickSearch() {
	p := a.GetActivePanel()

	input := tview.NewInputField()
	input.SetLabel(" Search: ")
	input.SetFieldWidth(30)
	input.SetBackgroundColor(theme.ColorDialogBg)
	input.SetFieldBackgroundColor(tcell.NewRGBColor(0, 0, 128))
	input.SetFieldTextColor(tcell.ColorWhite)
	input.SetLabelColor(tcell.ColorYellow)
	input.SetBorder(true)
	input.SetBorderColor(theme.ColorDialogBorder)
	input.SetTitle(" Search ")

	input.SetChangedFunc(func(text string) {
		if text == "" {
			return
		}
		s := strings.ToLower(text)
		for i, e := range p.Entries {
			if strings.HasPrefix(strings.ToLower(e.Name), s) {
				p.SetCursor(i)
				p.Render()
				break
			}
		}
	})

	input.SetDoneFunc(func(key tcell.Key) {
		a.closeDialog("search")
	})

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(input, 44, 0, true).
			AddItem(nil, 0, 1, false),
			3, 0, true).
		AddItem(nil, 0, 1, false)

	a.showDialog("search", flex)
}

// FocusCmdLine switches focus to the command line and types the initial character.
func (a *App) FocusCmdLine(ch rune) {
	a.CmdLineFocused = true
	a.CmdLine.SetText(string(ch))
	a.TviewApp.SetFocus(a.CmdLine)
}

// ExecuteCommand runs a shell command.
func (a *App) ExecuteCommand(cmd string) {
	if cmd == "" {
		return
	}

	p := a.GetActivePanel()

	if p.IsRemote() {
		a.showRemoteError("Execute command")
		return
	}

	a.TviewApp.Suspend(func() {
		c := exec.Command("sh", "-c", cmd)
		c.Dir = p.Path
		c.Stdin = os.Stdin
		c.Stdout = os.Stdout
		c.Stderr = os.Stderr
		c.Run()

		os.Stdout.WriteString("\nPress Enter to continue...")
		buf := make([]byte, 1)
		for {
			os.Stdin.Read(buf)
			if buf[0] == '\n' || buf[0] == '\r' {
				break
			}
		}
	})

	p.Refresh()
	a.GetInactivePanel().Refresh()
	a.CmdLine.SetPath(p.Path)
}

// ActivateMenu activates the top menu bar.
func (a *App) ActivateMenu() {
	a.MenuActive = true
	a.MenuBar.Active = true
	a.showMenuDropdown()
}

// DeactivateMenu hides the menu.
func (a *App) DeactivateMenu() {
	saved := a.activePanel
	a.MenuActive = false
	a.MenuBar.Active = false
	a.Pages.RemovePage("dropdown")
	a.activePanel = saved
	a.focusActiveTable()
	a.updatePanelStates()
}

// showMenuDropdown builds a tview.List for the current menu and shows it.
func (a *App) showMenuDropdown() {
	panelDefs := func(p *panel.Panel) *menu.MenuDefs {
		return &menu.MenuDefs{
			OnBriefMode: func() { p.Mode = panel.ModeBrief; a.SaveConfig(); a.DeactivateMenu() },
			OnFullMode:  func() { p.Mode = panel.ModeFull; a.SaveConfig(); a.DeactivateMenu() },
			OnSortName:  func() { a.setSortModeOn(p, panel.SortByName); a.DeactivateMenu() },
			OnSortExt:   func() { a.setSortModeOn(p, panel.SortByExtension); a.DeactivateMenu() },
			OnSortSize:  func() { a.setSortModeOn(p, panel.SortBySize); a.DeactivateMenu() },
			OnSortTime:  func() { a.setSortModeOn(p, panel.SortByTime); a.DeactivateMenu() },
			OnCopy:      func() { a.DeactivateMenu(); a.CopyFiles() },
			OnMove:      func() { a.DeactivateMenu(); a.MoveFiles() },
			OnMkDir:     func() { a.DeactivateMenu(); a.MakeDir() },
			OnDelete:    func() { a.DeactivateMenu(); a.DeleteFiles() },
			OnQuit:      func() { a.TviewApp.Stop() },
			OnSwapPanels: func() { a.DeactivateMenu(); a.swapPanels() },
			OnRefresh:   func() { a.DeactivateMenu(); a.GetActivePanel().Refresh(); a.GetInactivePanel().Refresh() },
			OnViewFile:  func() { a.DeactivateMenu(); a.ViewFile() },
			OnEditFile:  func() { a.DeactivateMenu(); a.EditFile() },
		}
	}

	var items []menu.MenuItem
	switch a.MenuBar.Selected {
	case 0:
		items = menu.LeftMenuItems(panelDefs(a.LeftPanel))
	case 1:
		items = menu.FileMenuItems(panelDefs(a.GetActivePanel()))
	case 2:
		items = menu.CommandsMenuItems(panelDefs(a.GetActivePanel()))
	case 3:
		items = menu.RightMenuItems(panelDefs(a.RightPanel))
	}

	if len(items) == 0 {
		return
	}

	// Build a tview.List for the dropdown
	list := tview.NewList()
	list.ShowSecondaryText(false)
	list.SetBackgroundColor(theme.ColorDialogBg)
	list.SetMainTextColor(theme.ColorDialogFg)
	list.SetSelectedTextColor(tcell.ColorBlack)
	list.SetSelectedBackgroundColor(tcell.NewRGBColor(0, 170, 170))
	list.SetHighlightFullLine(true)
	list.SetBorder(true)
	list.SetBorderColor(theme.ColorDialogBorder)
	list.SetTitle("")

	maxWidth := 0
	for _, item := range items {
		label := item.Label
		if item.Key != "" {
			label += "  " + item.Key
		}
		if len(label) > maxWidth {
			maxWidth = len(label)
		}

		if item.IsSep {
			list.AddItem("────────────────", "", 0, nil)
		} else {
			action := item.Action
			list.AddItem(label, "", 0, func() {
				if action != nil {
					action()
				}
			})
		}
	}

	// Handle keys within the list
	list.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyLeft:
			a.MenuBar.MoveLeft()
			a.showMenuDropdown()
			return nil
		case tcell.KeyRight:
			a.MenuBar.MoveRight()
			a.showMenuDropdown()
			return nil
		case tcell.KeyEscape, tcell.KeyF9:
			a.DeactivateMenu()
			return nil
		case tcell.KeyF10:
			a.TviewApp.Stop()
			return nil
		}
		return event
	})

	// Calculate position
	xPos := 0
	for i := 0; i < a.MenuBar.Selected; i++ {
		xPos += len(a.MenuBar.Items[i])
	}

	width := maxWidth + 4
	if width < 16 {
		width = 16
	}
	height := len(items) + 2

	list.SetRect(xPos, 1, width, height)

	a.Pages.RemovePage("dropdown")
	a.Pages.AddPage("dropdown", list, false, true) // false = don't resize to fullscreen
	a.TviewApp.SetFocus(list)
}

func (a *App) setSortMode(mode panel.SortMode) {
	a.setSortModeOn(a.GetActivePanel(), mode)
}

func (a *App) setSortModeOn(p *panel.Panel, mode panel.SortMode) {
	p.SortMode = mode
	panel.SortEntries(p.Entries, mode)
	p.SetCursor(0)
	p.Render()
	a.SaveConfig()
}

// SaveConfig persists panel display modes and sort modes.
func (a *App) SaveConfig() {
	config.Save(&config.Config{
		LeftPanel:  config.PanelConfig{Mode: int(a.LeftPanel.Mode), SortMode: int(a.LeftPanel.SortMode)},
		RightPanel: config.PanelConfig{Mode: int(a.RightPanel.Mode), SortMode: int(a.RightPanel.SortMode)},
	})
}

func (a *App) swapPanels() {
	a.LeftPanel.Path, a.RightPanel.Path = a.RightPanel.Path, a.LeftPanel.Path
	a.LeftPanel.FS, a.RightPanel.FS = a.RightPanel.FS, a.LeftPanel.FS
	a.LeftPanel.ConnectedServer, a.RightPanel.ConnectedServer = a.RightPanel.ConnectedServer, a.LeftPanel.ConnectedServer
	a.LeftPanel.Refresh()
	a.RightPanel.Refresh()
}

// ShowServerDialog opens the F1 server connection dialog.
func (a *App) ShowServerDialog() {
	cfg := config.Load()

	var showDialog func()
	showDialog = func() {
		dialog.ShowServerDialog(a.Pages, cfg.Servers, dialog.ServerDialogCallbacks{
			OnConnect: func(srv config.ServerConfig) {
				a.closeDialog("server_dialog")
				a.connectPanel(a.GetActivePanel(), srv)
			},
			OnDisconnect: func(name string) {
				a.closeDialog("server_dialog")
				p := a.GetActivePanel()
				if p.ConnectedServer == name {
					a.disconnectPanel(p)
				}
			},
			OnAdd: func() {
				a.Pages.RemovePage("server_dialog")
				dialog.ShowServerEdit(a.Pages, "Add Server", config.ServerConfig{Protocol: "sftp"}, func(srv config.ServerConfig) {
					a.Pages.RemovePage("server_edit")
					cfg.Servers = append(cfg.Servers, srv)
					a.saveConfigWithServers(cfg)
					showDialog()
				}, func() {
					a.Pages.RemovePage("server_edit")
					a.ModalOpen = false
					showDialog()
				})
			},
			OnEdit: func(idx int, srv config.ServerConfig) {
				a.Pages.RemovePage("server_dialog")
				dialog.ShowServerEdit(a.Pages, "Edit Server", srv, func(updated config.ServerConfig) {
					a.Pages.RemovePage("server_edit")
					cfg.Servers[idx] = updated
					a.saveConfigWithServers(cfg)
					showDialog()
				}, func() {
					a.Pages.RemovePage("server_edit")
					a.ModalOpen = false
					showDialog()
				})
			},
			OnDelete: func(idx int) {
				a.closeDialog("server_dialog")
				name := cfg.Servers[idx].Name
				dialog.ShowConfirm(a.Pages, "Delete Server", "Delete server '"+name+"'?", func(yes bool) {
					a.closeDialog("confirm")
					if yes {
						cfg.Servers = append(cfg.Servers[:idx], cfg.Servers[idx+1:]...)
						a.saveConfigWithServers(cfg)
					}
					showDialog()
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
			},
			OnClose: func() {
				a.closeDialog("server_dialog")
			},
			IsConnected: func(name string) bool {
				return a.ConnMgr.IsConnected(name)
			},
		})
		a.ModalOpen = true
		a.TviewApp.SetFocus(a.Pages)
	}
	showDialog()
}

func (a *App) connectPanel(p *panel.Panel, srv config.ServerConfig) {
	// Show a simple "connecting" message
	dialog.ShowError(a.Pages, "Connecting to "+srv.Name+"...", nil)
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
	a.TviewApp.ForceDraw()

	go func() {
		fs, err := a.ConnMgr.Connect(srv)
		a.TviewApp.QueueUpdateDraw(func() {
			a.closeDialog("error")
			if err != nil {
				dialog.ShowError(a.Pages, "Connection failed: "+err.Error(), func() {
					a.closeDialog("error")
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
				return
			}

			p.FS = fs
			p.ConnectedServer = srv.Name
			p.Path = "/"
			p.Refresh()
			a.CmdLine.SetPath(a.GetActivePanel().Path)
		})
	}()
}

func (a *App) disconnectPanel(p *panel.Panel) {
	name := p.ConnectedServer
	if name == "" {
		return
	}

	// Check if the other panel uses the same connection
	other := a.GetInactivePanel()
	otherUsesSame := other.ConnectedServer == name

	p.FS = vfs.NewLocalFS()
	p.ConnectedServer = ""
	home, err := os.UserHomeDir()
	if err != nil {
		home = "/"
	}
	p.Path = home
	p.Refresh()
	a.CmdLine.SetPath(a.GetActivePanel().Path)

	if !otherUsesSame {
		a.ConnMgr.Disconnect(name)
	}
}

func (a *App) saveConfigWithServers(cfg *config.Config) {
	cfg.LeftPanel = config.PanelConfig{Mode: int(a.LeftPanel.Mode), SortMode: int(a.LeftPanel.SortMode)}
	cfg.RightPanel = config.PanelConfig{Mode: int(a.RightPanel.Mode), SortMode: int(a.RightPanel.SortMode)}
	config.Save(cfg)
}

func (a *App) showRemoteError(operation string) {
	dialog.ShowError(a.Pages, operation+" is not available on remote panels.", func() {
		a.closeDialog("error")
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
}

func entryNames(entries []model.FileEntry) string {
	if len(entries) == 1 {
		return entries[0].Name
	}
	return fmt.Sprintf("%d files/dirs", len(entries))
}
