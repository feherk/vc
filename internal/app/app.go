package app

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"golang.org/x/crypto/argon2"

	"github.com/feherkaroly/vc/internal/cmdline"
	"github.com/feherkaroly/vc/internal/config"
	"github.com/feherkaroly/vc/internal/dialog"
	"github.com/feherkaroly/vc/internal/fileops"
	"github.com/feherkaroly/vc/internal/fnbar"
	"github.com/feherkaroly/vc/internal/menu"
	"github.com/feherkaroly/vc/internal/model"
	"github.com/feherkaroly/vc/internal/panel"
	"github.com/feherkaroly/vc/internal/platform"
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

	lastClickTime  time.Time
	lastClickRow   int
	lastClickTable *tview.Table

	activeDropdown *menu.Dropdown
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

	a.activePanel = cfg.ActivePanel
	if a.activePanel == 1 {
		a.LeftPanel.SetActive(false)
		a.RightPanel.SetActive(true)
	} else {
		a.activePanel = 0
		a.LeftPanel.SetActive(true)
		a.RightPanel.SetActive(false)
	}

	a.TviewApp.EnableMouse(true)

	a.MenuBar.OnClick = func(idx int) {
		if a.ModalOpen {
			return
		}
		a.MenuBar.Selected = idx
		a.ActivateMenu()
		go a.TviewApp.QueueUpdateDraw(func() {})
	}

	a.FnBar.OnClick = func(fn int) {
		if a.ModalOpen || a.MenuActive {
			return
		}
		switch fn {
		case 1:
			a.ShowServerDialog()
		case 2:
			a.CompressFiles()
		case 3:
			a.ViewFile()
		case 4:
			a.EditFile()
		case 5:
			a.CopyFiles()
		case 6:
			a.MoveFiles()
		case 7:
			a.MakeDir()
		case 8:
			a.DeleteFiles()
		case 9:
			a.ActivateMenu()
		case 10:
			a.SaveConfig()
			a.TviewApp.Stop()
		case 11:
			a.GetActivePanel().ToggleSelection()
		}
		go a.TviewApp.QueueUpdateDraw(func() {})
	}

	a.SetupKeyBindings()
	a.TviewApp.SetFocus(a.activeTable())

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
	saved := a.activePanel
	a.TviewApp.SetRoot(a.Pages, true)
	a.activePanel = saved
	a.TviewApp.SetFocus(a.activeTable())
	a.updatePanelStates()
	return a.TviewApp.Run()
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

// CompressFiles handles F2 — compress selected items into zip/tar/tar.gz archive,
// or encrypt/decrypt a single file.
func (a *App) CompressFiles() {
	p := a.GetActivePanel()
	if p.IsRemote() {
		a.showRemoteError("Compress")
		return
	}
	entries := p.GetSelectedOrCurrent()
	if len(entries) == 0 {
		return
	}

	desc := entryNames(entries)

	singleFile := len(entries) == 1 && !entries[0].IsDir
	isEnc := singleFile && strings.HasSuffix(strings.ToLower(entries[0].Name), ".enc")
	lower := strings.ToLower(entries[0].Name)
	isArchive := singleFile && (strings.HasSuffix(lower, ".zip") ||
		strings.HasSuffix(lower, ".tar.gz") || strings.HasSuffix(lower, ".tgz") ||
		strings.HasSuffix(lower, ".tar"))

	dialog.ShowFormatDialog(a.Pages, singleFile, isEnc, isArchive, func(format string) {
		a.closeDialog("format")

		if format == "extract" {
			srcPath := filepath.Join(p.Path, entries[0].Name)
			destDir := uniqueExtractDir(p.Path, entries[0].Name)
			a.runWithSpinner(entries[0].Name, func() error {
				if err := os.MkdirAll(destDir, 0755); err != nil {
					return err
				}
				lower := strings.ToLower(entries[0].Name)
				if strings.HasSuffix(lower, ".zip") {
					return extractZip(srcPath, destDir)
				}
				return extractTar(srcPath, destDir)
			})
			return
		}

		if format == "encrypt" {
			srcPath := filepath.Join(p.Path, entries[0].Name)
			dialog.ShowPasswordDialog(a.Pages, "Encrypt", true, func(password string) {
				a.closeDialog("password")
				dstName := fmt.Sprintf("enc_%d.enc", time.Now().Unix())
				dstPath := filepath.Join(p.Path, dstName)
				a.runWithSpinner(dstName, func() error {
					return encryptFile(srcPath, dstPath, password)
				})
			}, func() {
				a.closeDialog("password")
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
			return
		}

		if format == "decrypt" {
			srcPath := filepath.Join(p.Path, entries[0].Name)
			dialog.ShowPasswordDialog(a.Pages, "Decrypt", false, func(password string) {
				a.closeDialog("password")
				a.runWithSpinner(entries[0].Name, func() error {
					_, err := decryptFile(srcPath, p.Path, password)
					return err
				})
			}, func() {
				a.closeDialog("password")
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
			return
		}

		var ext string
		switch format {
		case "tar":
			ext = ".tar"
		case "tar.gz":
			ext = ".tar.gz"
		default:
			ext = ".zip"
		}

		var baseName string
		if p.Selection.Count() > 0 {
			baseName = fmt.Sprintf("archiv_%d", time.Now().UnixNano())
		} else {
			baseName = entries[0].Name
			if e := filepath.Ext(baseName); e != "" {
				baseName = strings.TrimSuffix(baseName, e)
			}
		}
		archiveName := baseName + ext

		dialog.ShowInput(a.Pages, "Compress", "Compress "+desc+" to:", archiveName, func(name string) {
			a.closeDialog("input")
			if name == "" {
				return
			}

			srcDir := p.Path
			archivePath := filepath.Join(srcDir, name)

			// Non-modal spinner in bottom-right corner
			spinView := tview.NewTextView()
			spinView.SetBackgroundColor(theme.ColorDialogBg)
			spinView.SetTextColor(theme.ColorDialogFg)
			spinView.SetBorder(true)
			spinView.SetBorderColor(theme.ColorDialogBorder)
			spinView.SetTextAlign(tview.AlignCenter)

			displayName := name
			if len(displayName) > 20 {
				displayName = displayName[:20] + "..."
			}
			boxW := len(displayName) + 6
			if boxW < 18 {
				boxW = 18
			}
			_, _, screenW, screenH := a.Pages.GetInnerRect()
			spinView.SetRect(screenW-boxW-1, screenH-4, boxW, 3)
			spinView.SetText(displayName + " |")

			a.Pages.AddPage("spinner", spinView, false, true)
			a.focusActiveTable()

			go func() {
				spinChars := [4]rune{'|', '/', '-', '\\'}
				spinIdx := 0
				done := make(chan struct{})

				go func() {
					ticker := time.NewTicker(150 * time.Millisecond)
					defer ticker.Stop()
					for {
						select {
						case <-done:
							return
						case <-ticker.C:
							spinIdx = (spinIdx + 1) % 4
							ch := spinChars[spinIdx]
							a.TviewApp.QueueUpdateDraw(func() {
								spinView.SetText(fmt.Sprintf("%s %c", displayName, ch))
							})
						}
					}
				}()

				var err error
				switch format {
				case "tar":
					err = createTar(archivePath, srcDir, entries, false)
				case "tar.gz":
					err = createTar(archivePath, srcDir, entries, true)
				default:
					err = createZip(archivePath, srcDir, entries)
				}
				close(done)

				a.TviewApp.QueueUpdateDraw(func() {
					saved := a.activePanel
					a.Pages.RemovePage("spinner")
					a.activePanel = saved
					a.focusActiveTable()
					a.updatePanelStates()
					if err != nil {
						dialog.ShowError(a.Pages, "Compress error: "+err.Error(), func() {
							a.closeDialog("error")
						})
						a.ModalOpen = true
						a.TviewApp.SetFocus(a.Pages)
						return
					}
					p.Selection.Clear()
					p.Refresh()
					a.GetInactivePanel().Refresh()
				})
			}()
		}, func() {
			a.closeDialog("input")
		})
		a.ModalOpen = true
		a.TviewApp.SetFocus(a.Pages)
	}, func() {
		a.closeDialog("format")
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
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

// createTar creates a tar (or tar.gz if compress=true) archive at tarPath.
func createTar(tarPath, baseDir string, entries []model.FileEntry, compress bool) error {
	f, err := os.Create(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	var tw *tar.Writer
	if compress {
		gw := gzip.NewWriter(f)
		defer gw.Close()
		tw = tar.NewWriter(gw)
	} else {
		tw = tar.NewWriter(f)
	}
	defer tw.Close()

	for _, entry := range entries {
		srcPath := filepath.Join(baseDir, entry.Name)
		if entry.IsDir {
			err = addDirToTar(tw, srcPath, entry.Name)
		} else {
			err = addFileToTar(tw, srcPath, entry.Name)
		}
		if err != nil {
			return err
		}
	}
	return nil
}

func addFileToTar(tw *tar.Writer, filePath, nameInTar string) error {
	f, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer f.Close()

	info, err := f.Stat()
	if err != nil {
		return err
	}

	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = nameInTar

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err = io.Copy(tw, f)
	return err
}

func addDirToTar(tw *tar.Writer, dirPath, prefix string) error {
	return filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.Mode()&os.ModeSymlink != 0 {
			return nil
		}

		rel, err := filepath.Rel(filepath.Dir(dirPath), path)
		if err != nil {
			return err
		}

		if info.IsDir() {
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = rel + "/"
			return tw.WriteHeader(header)
		}

		if !info.Mode().IsRegular() {
			return nil
		}

		return addFileToTar(tw, path, rel)
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
	switch runtime.GOOS {
	case "darwin":
		exec.Command("open", path).Start()
	case "windows":
		exec.Command("cmd", "/c", "start", "", path).Start()
	default:
		exec.Command("xdg-open", path).Start()
	}
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

		a.runWithSpinner("Copying", func() error {
			for _, entry := range entries {
				srcPath := srcFS.Join(src.Path, entry.Name)
				dstPath := dstFS.Join(target, entry.Name)
				if err := fileops.Copy(context.Background(), srcFS, srcPath, dstFS, dstPath, nil); err != nil {
					return err
				}
			}
			return nil
		})
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

		a.runWithSpinner("Moving", func() error {
			for _, entry := range entries {
				srcPath := srcFS.Join(src.Path, entry.Name)
				dstPath := target
				if len(entries) > 1 {
					dstPath = dstFS.Join(target, entry.Name)
				}
				if err := fileops.Move(context.Background(), srcFS, srcPath, dstFS, dstPath, nil); err != nil {
					return err
				}
			}
			return nil
		})
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

// CreateSymlink creates symbolic links in the inactive panel for selected entries.
func (a *App) CreateSymlink() {
	src := a.GetActivePanel()
	dst := a.GetInactivePanel()

	if src.IsRemote() || dst.IsRemote() {
		a.showRemoteError("Symlink")
		return
	}

	entries := src.GetSelectedOrCurrent()
	if len(entries) == 0 {
		return
	}

	createLinks := func(names map[string]string) {
		for origName, linkName := range names {
			target := filepath.Join(src.Path, origName)
			linkPath := filepath.Join(dst.Path, linkName)
			if err := os.Symlink(target, linkPath); err != nil {
				dialog.ShowError(a.Pages, "Symlink error: "+err.Error(), func() {
					a.closeDialog("error")
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
				return
			}
		}
		src.Selection.Clear()
		src.Refresh()
		dst.Refresh()
	}

	if len(entries) == 1 {
		dialog.ShowInput(a.Pages, "Symlink", "Create symlink as:", entries[0].Name, func(name string) {
			a.closeDialog("input")
			if name == "" {
				return
			}
			createLinks(map[string]string{entries[0].Name: name})
		}, func() {
			a.closeDialog("input")
		})
		a.ModalOpen = true
		a.TviewApp.SetFocus(a.Pages)
		return
	}

	// Multiple entries: create symlinks with original names
	names := make(map[string]string, len(entries))
	for _, e := range entries {
		names[e.Name] = e.Name
	}
	createLinks(names)
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

		a.runWithSpinner(desc, func() error {
			for _, entry := range entries {
				path := p.FS.Join(p.Path, entry.Name)
				if err := fileops.Delete(p.FS, path); err != nil {
					return err
				}
			}
			return nil
		})
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
	input.SetFieldBackgroundColor(theme.ColorDialogBg)
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

	// Handle "cd" command: navigate the active panel (works on remote panels too)
	trimmed := strings.TrimSpace(cmd)
	if trimmed == "cd" || strings.HasPrefix(trimmed, "cd ") {
		dir := strings.TrimSpace(strings.TrimPrefix(trimmed, "cd"))
		if p.IsRemote() {
			if dir == "" || dir == "/" {
				dir = "/"
			} else if dir == ".." {
				dir = p.FS.Dir(p.Path)
			} else if !strings.HasPrefix(dir, "/") {
				dir = p.FS.Join(p.Path, dir)
			}
		} else {
			if dir == "" || dir == "~" {
				dir, _ = os.UserHomeDir()
			} else if strings.HasPrefix(dir, "~/") {
				home, _ := os.UserHomeDir()
				dir = filepath.Join(home, dir[2:])
			} else if !filepath.IsAbs(dir) {
				dir = filepath.Join(p.Path, dir)
			}
			dir = filepath.Clean(dir)
		}
		p.NavigateTo(dir, "")
		a.CmdLine.SetPath(p.Path)
		return
	}

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
	a.activeDropdown = nil
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
			OnQuit:      func() { a.SaveConfig(); a.TviewApp.Stop() },
			OnSwapPanels: func() { a.DeactivateMenu(); a.swapPanels() },
			OnRefresh:   func() { a.DeactivateMenu(); a.GetActivePanel().Refresh(); a.GetInactivePanel().Refresh() },
			OnViewFile:     func() { a.DeactivateMenu(); a.ViewFile() },
			OnEditFile:     func() { a.DeactivateMenu(); a.EditFile() },
			OnExportConfig: func() { a.DeactivateMenu(); a.ExportConfig() },
			OnImportConfig: func() { a.DeactivateMenu(); a.ImportConfig() },
			OnQuickPaths:   func() { a.DeactivateMenu(); a.ShowQuickPathsDialog() },
			OnCheckUpdate:  func() { a.DeactivateMenu(); a.CheckForUpdates() },
			OnSymlink:      func() { a.DeactivateMenu(); a.CreateSymlink() },
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

	dd := menu.NewDropdown()
	dd.SetItems(items)
	dd.Visible = true

	// Calculate position
	xPos := 0
	for i := 0; i < a.MenuBar.Selected; i++ {
		xPos += len(a.MenuBar.Items[i])
	}
	dd.X = xPos
	dd.Y = 1

	// Calculate size for tview rect
	maxWidth := 0
	for _, item := range items {
		w := len(item.Label) + len(item.Key) + 4
		if w > maxWidth {
			maxWidth = w
		}
	}
	width := maxWidth + 2
	height := len(items) + 2
	dd.SetRect(xPos, 1, width, height)

	dd.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyDown:
			dd.MoveDown()
			return nil
		case tcell.KeyUp:
			dd.MoveUp()
			return nil
		case tcell.KeyEnter:
			action := dd.CurrentAction()
			if action != nil {
				action()
			}
			return nil
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
			a.SaveConfig()
			a.TviewApp.Stop()
			return nil
		}
		return event
	})

	a.Pages.RemovePage("dropdown")
	a.activeDropdown = dd
	a.Pages.AddPage("dropdown", dd, false, true)
	a.TviewApp.SetFocus(dd)
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

// SaveConfig persists panel display modes, sort modes, and paths.
// It preserves existing servers from the config file.
func (a *App) SaveConfig() {
	cfg := config.Load()
	a.saveConfigWithServers(cfg)
}

// ExportConfig exports the full config (panels + servers) to a user-specified file.
func (a *App) ExportConfig() {
	p := a.GetActivePanel()
	defaultPath := filepath.Join(p.Path, "vc-config.json")
	dialog.ShowInput(a.Pages, "Export config", "Export config to:", defaultPath, func(target string) {
		a.closeDialog("input")
		if target == "" {
			return
		}

		cfg := config.Load()
		data, err := json.MarshalIndent(cfg, "", "  ")
		if err != nil {
			dialog.ShowError(a.Pages, "Export error: "+err.Error(), func() {
				a.closeDialog("error")
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
			return
		}

		if err := os.WriteFile(target, data, 0600); err != nil {
			dialog.ShowError(a.Pages, "Export error: "+err.Error(), func() {
				a.closeDialog("error")
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
			return
		}

		a.GetActivePanel().Refresh()
		a.GetInactivePanel().Refresh()
	}, func() {
		a.closeDialog("input")
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
}

// ImportConfig imports config from a user-specified file, merging servers.
func (a *App) ImportConfig() {
	dialog.ShowInput(a.Pages, "Import config", "Import config from:", "vc-config.json", func(target string) {
		a.closeDialog("input")
		if target == "" {
			return
		}

		data, err := os.ReadFile(target)
		if err != nil {
			dialog.ShowError(a.Pages, "Import error: "+err.Error(), func() {
				a.closeDialog("error")
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
			return
		}

		var imported config.Config
		if err := json.Unmarshal(data, &imported); err != nil {
			dialog.ShowError(a.Pages, "Import error: "+err.Error(), func() {
				a.closeDialog("error")
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
			return
		}

		// Apply panel settings
		a.LeftPanel.Mode = panel.DisplayMode(imported.LeftPanel.Mode)
		a.LeftPanel.SortMode = panel.SortMode(imported.LeftPanel.SortMode)
		if imported.LeftPanel.Path != "" {
			a.LeftPanel.NavigateTo(imported.LeftPanel.Path, "")
		}
		a.RightPanel.Mode = panel.DisplayMode(imported.RightPanel.Mode)
		a.RightPanel.SortMode = panel.SortMode(imported.RightPanel.SortMode)
		if imported.RightPanel.Path != "" {
			a.RightPanel.NavigateTo(imported.RightPanel.Path, "")
		}

		// Merge servers: imported overwrites existing by name, new ones are appended
		current := config.Load()
		serverMap := make(map[string]config.ServerConfig)
		for _, s := range current.Servers {
			serverMap[s.Name] = s
		}
		for _, s := range imported.Servers {
			serverMap[s.Name] = s
		}
		merged := make([]config.ServerConfig, 0, len(serverMap))
		for _, s := range serverMap {
			merged = append(merged, s)
		}

		a.saveConfigWithServers(&config.Config{Servers: merged})
		a.LeftPanel.Refresh()
		a.RightPanel.Refresh()
		a.CmdLine.SetPath(a.GetActivePanel().Path)
	}, func() {
		a.closeDialog("input")
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
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
	leftPath := ""
	if a.LeftPanel.FS.IsLocal() {
		leftPath = a.LeftPanel.Path
	}
	rightPath := ""
	if a.RightPanel.FS.IsLocal() {
		rightPath = a.RightPanel.Path
	}
	cfg.LeftPanel = config.PanelConfig{Mode: int(a.LeftPanel.Mode), SortMode: int(a.LeftPanel.SortMode), Path: leftPath}
	cfg.RightPanel = config.PanelConfig{Mode: int(a.RightPanel.Mode), SortMode: int(a.RightPanel.SortMode), Path: rightPath}
	cfg.ActivePanel = a.activePanel
	config.Save(cfg)
}

// ShowDriveSelector shows a drive selection dialog (Windows only).
func (a *App) ShowDriveSelector() {
	drives := platform.GetDrives()
	if len(drives) == 0 {
		return
	}

	p := a.GetActivePanel()
	dialog.ShowDriveDialog(a.Pages, drives, func(drive string) {
		a.closeDialog("drive_dialog")
		p.NavigateTo(drive, "")
		a.CmdLine.SetPath(p.Path)
	}, func() {
		a.closeDialog("drive_dialog")
	})
	a.ModalOpen = true
	a.TviewApp.SetFocus(a.Pages)
}

// ShowQuickPathsDialog opens the quick paths (1-9) dialog.
func (a *App) ShowQuickPathsDialog() {
	cfg := config.Load()
	if cfg.QuickPaths == nil {
		cfg.QuickPaths = make(map[string]string)
	}

	var showDialog func()
	showDialog = func() {
		dialog.ShowQuickPathsDialog(a.Pages, cfg.QuickPaths, dialog.QuickPathsCallbacks{
			OnSet: func(slot int) {
				key := fmt.Sprintf("%d", slot)
				cfg.QuickPaths[key] = a.GetActivePanel().Path
				a.saveConfigWithServers(cfg)
				a.Pages.RemovePage("quickpaths")
				a.ModalOpen = false
				showDialog()
			},
			OnEdit: func(slot int, current string) {
				a.Pages.RemovePage("quickpaths")
				key := fmt.Sprintf("%d", slot)
				dialog.ShowInput(a.Pages, fmt.Sprintf("Quick Path %d", slot), "Path:", current, func(path string) {
					a.closeDialog("input")
					if path != "" {
						cfg.QuickPaths[key] = path
					} else {
						delete(cfg.QuickPaths, key)
					}
					a.saveConfigWithServers(cfg)
					showDialog()
				}, func() {
					a.closeDialog("input")
					showDialog()
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
			},
			OnDelete: func(slot int) {
				key := fmt.Sprintf("%d", slot)
				delete(cfg.QuickPaths, key)
				a.saveConfigWithServers(cfg)
				a.Pages.RemovePage("quickpaths")
				a.ModalOpen = false
				showDialog()
			},
			OnGo: func(slot int) {
				a.closeDialog("quickpaths")
				a.navigateToQuickPath(slot, cfg.QuickPaths)
			},
			OnClose: func() {
				a.closeDialog("quickpaths")
			},
		})
		a.ModalOpen = true
		a.TviewApp.SetFocus(a.Pages)
	}
	showDialog()
}

// navigateToQuickPath navigates the active panel to the path stored in the given slot.
func (a *App) navigateToQuickPath(slot int, paths map[string]string) {
	key := fmt.Sprintf("%d", slot)
	path, ok := paths[key]
	if !ok || path == "" {
		return
	}
	p := a.GetActivePanel()
	p.NavigateTo(path, "")
	a.CmdLine.SetPath(p.Path)
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

// runWithSpinner runs fn in a goroutine with a non-modal spinner, then refreshes panels.
func (a *App) runWithSpinner(displayName string, fn func() error) {
	p := a.GetActivePanel()

	spinView := tview.NewTextView()
	spinView.SetBackgroundColor(theme.ColorDialogBg)
	spinView.SetTextColor(theme.ColorDialogFg)
	spinView.SetBorder(true)
	spinView.SetBorderColor(theme.ColorDialogBorder)
	spinView.SetTextAlign(tview.AlignCenter)

	if len(displayName) > 20 {
		displayName = displayName[:20] + "..."
	}
	boxW := len(displayName) + 6
	if boxW < 18 {
		boxW = 18
	}
	_, _, screenW, screenH := a.Pages.GetInnerRect()
	spinView.SetRect(screenW-boxW-1, screenH-4, boxW, 3)
	spinView.SetText(displayName + " |")

	a.Pages.AddPage("spinner", spinView, false, true)
	a.focusActiveTable()

	go func() {
		spinChars := [4]rune{'|', '/', '-', '\\'}
		spinIdx := 0
		done := make(chan struct{})

		go func() {
			ticker := time.NewTicker(150 * time.Millisecond)
			defer ticker.Stop()
			for {
				select {
				case <-done:
					return
				case <-ticker.C:
					spinIdx = (spinIdx + 1) % 4
					ch := spinChars[spinIdx]
					a.TviewApp.QueueUpdateDraw(func() {
						spinView.SetText(fmt.Sprintf("%s %c", displayName, ch))
					})
				}
			}
		}()

		err := fn()
		close(done)

		a.TviewApp.QueueUpdateDraw(func() {
			saved := a.activePanel
			a.Pages.RemovePage("spinner")
			a.activePanel = saved
			a.focusActiveTable()
			a.updatePanelStates()
			if err != nil {
				dialog.ShowError(a.Pages, "Error: "+err.Error(), func() {
					a.closeDialog("error")
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
				return
			}
			p.Selection.Clear()
			p.Refresh()
			a.GetInactivePanel().Refresh()
		})
	}()
}

// encryptFile encrypts srcPath with AES-256-GCM and writes to dstPath.
// File format: [2 byte filename length][filename][16 byte salt][12 byte nonce][ciphertext+tag]
func encryptFile(srcPath, dstPath, password string) error {
	plaintext, err := os.ReadFile(srcPath)
	if err != nil {
		return err
	}

	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return err
	}

	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return err
	}

	nonce := make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := rand.Read(nonce); err != nil {
		return err
	}

	origName := filepath.Base(srcPath)

	// Build header: filename length (uint16 big-endian) + filename
	var header []byte
	header = binary.BigEndian.AppendUint16(header, uint16(len(origName)))
	header = append(header, []byte(origName)...)

	// Header is included as AAD so any tampering with the filename is detected
	ciphertext := gcm.Seal(nil, nonce, plaintext, header)

	out, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Write header
	if _, err := out.Write(header); err != nil {
		return err
	}
	// Write salt + nonce + ciphertext
	if _, err := out.Write(salt); err != nil {
		return err
	}
	if _, err := out.Write(nonce); err != nil {
		return err
	}
	if _, err := out.Write(ciphertext); err != nil {
		return err
	}

	return nil
}

// decryptFile decrypts srcPath and writes the original file to dstDir with its original name.
// Returns the original filename.
func decryptFile(srcPath, dstDir, password string) (string, error) {
	data, err := os.ReadFile(srcPath)
	if err != nil {
		return "", err
	}

	if len(data) < 2 {
		return "", fmt.Errorf("invalid encrypted file")
	}

	nameLen := binary.BigEndian.Uint16(data[:2])
	offset := 2 + int(nameLen)

	if len(data) < offset+16+12 {
		return "", fmt.Errorf("invalid encrypted file")
	}

	origName := string(data[2:offset])
	salt := data[offset : offset+16]
	nonce := data[offset+16 : offset+16+12]
	ciphertext := data[offset+16+12:]

	key := argon2.IDKey([]byte(password), salt, 1, 64*1024, 4, 32)

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	header := data[:offset] // filename length + filename
	plaintext, err := gcm.Open(nil, nonce, ciphertext, header)
	if err != nil {
		return "", fmt.Errorf("decryption failed (wrong password?)")
	}

	dstPath := filepath.Join(dstDir, origName)
	if err := os.WriteFile(dstPath, plaintext, 0600); err != nil {
		return "", err
	}

	return origName, nil
}

// uniqueExtractDir returns a unique directory path for extracting an archive.
// It strips the archive extension and appends a number suffix if needed.
func uniqueExtractDir(parentDir, archiveName string) string {
	lower := strings.ToLower(archiveName)
	base := archiveName
	switch {
	case strings.HasSuffix(lower, ".tar.gz"):
		base = archiveName[:len(archiveName)-7]
	case strings.HasSuffix(lower, ".tgz"):
		base = archiveName[:len(archiveName)-4]
	case strings.HasSuffix(lower, ".zip"):
		base = archiveName[:len(archiveName)-4]
	case strings.HasSuffix(lower, ".tar"):
		base = archiveName[:len(archiveName)-4]
	}

	candidate := filepath.Join(parentDir, base)
	if _, err := os.Stat(candidate); os.IsNotExist(err) {
		return candidate
	}
	for i := 2; ; i++ {
		candidate = filepath.Join(parentDir, fmt.Sprintf("%s%d", base, i))
		if _, err := os.Stat(candidate); os.IsNotExist(err) {
			return candidate
		}
	}
}

// extractZip extracts a zip archive to destDir.
func extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return err
	}
	defer r.Close()

	for _, f := range r.File {
		target := filepath.Join(destDir, f.Name)
		// ZipSlip protection
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path in archive: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			os.MkdirAll(target, 0755)
			continue
		}

		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return err
		}

		rc, err := f.Open()
		if err != nil {
			return err
		}

		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}

		_, err = io.Copy(out, rc)
		rc.Close()
		out.Close()
		if err != nil {
			return err
		}
	}
	return nil
}

// extractTar extracts a tar (or tar.gz/tgz) archive to destDir.
// Gzip detection is based on the file's magic bytes.
func extractTar(tarPath, destDir string) error {
	f, err := os.Open(tarPath)
	if err != nil {
		return err
	}
	defer f.Close()

	// Detect gzip by reading magic bytes
	magic := make([]byte, 2)
	if _, err := io.ReadFull(f, magic); err != nil {
		return err
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}

	var tr *tar.Reader
	if magic[0] == 0x1f && magic[1] == 0x8b {
		gr, err := gzip.NewReader(f)
		if err != nil {
			return err
		}
		defer gr.Close()
		tr = tar.NewReader(gr)
	} else {
		tr = tar.NewReader(f)
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)
		// ZipSlip protection
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid path in archive: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0755)
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			out.Close()
			if err != nil {
				return err
			}
		}
	}
	return nil
}
