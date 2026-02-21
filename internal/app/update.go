package app

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/dialog"
	"github.com/feherkaroly/vc/internal/theme"
)

type ghRelease struct {
	TagName string    `json:"tag_name"`
	Assets  []ghAsset `json:"assets"`
}

type ghAsset struct {
	Name               string `json:"name"`
	BrowserDownloadURL string `json:"browser_download_url"`
}

// CheckForUpdates queries the latest GitHub release and offers to update if newer.
func (a *App) CheckForUpdates() {
	spinner := a.showSpinner("Checking...")

	go func() {
		rel, err := fetchLatestRelease()
		a.removeSpinner(spinner)

		if err != nil {
			a.TviewApp.QueueUpdateDraw(func() {
				dialog.ShowError(a.Pages, "Cannot check for updates: "+err.Error(), func() {
					a.closeDialog("error")
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
			})
			return
		}

		latest := strings.TrimPrefix(rel.TagName, "v")
		current := Version

		if !isNewer(current, latest) {
			a.TviewApp.QueueUpdateDraw(func() {
				dialog.ShowError(a.Pages, fmt.Sprintf("Already up to date (v%s)", current), func() {
					a.closeDialog("error")
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
			})
			return
		}

		// Find the right asset
		assetName := fmt.Sprintf("vc-%s-%s", runtime.GOOS, runtime.GOARCH)
		var downloadURL string
		for _, asset := range rel.Assets {
			if asset.Name == assetName {
				downloadURL = asset.BrowserDownloadURL
				break
			}
		}
		if downloadURL == "" {
			a.TviewApp.QueueUpdateDraw(func() {
				dialog.ShowError(a.Pages, fmt.Sprintf("No download available for %s/%s", runtime.GOOS, runtime.GOARCH), func() {
					a.closeDialog("error")
				})
				a.ModalOpen = true
				a.TviewApp.SetFocus(a.Pages)
			})
			return
		}

		// Ask user to confirm
		a.TviewApp.QueueUpdateDraw(func() {
			msg := fmt.Sprintf("Update available: v%s → v%s\nUpdate now?", current, latest)
			dialog.ShowConfirm(a.Pages, "Update", msg, func(yes bool) {
				a.closeDialog("confirm")
				if !yes {
					return
				}
				a.doUpdate(downloadURL, latest)
			})
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
		})
	}()
}

func (a *App) doUpdate(downloadURL, latest string) {
	spinner := a.showSpinner("Updating...")

	go func() {
		err := downloadAndReplace(downloadURL)
		a.removeSpinner(spinner)

		a.TviewApp.QueueUpdateDraw(func() {
			if err != nil {
				dialog.ShowError(a.Pages, "Update failed: "+err.Error(), func() {
					a.closeDialog("error")
				})
			} else {
				dialog.ShowError(a.Pages, fmt.Sprintf("Updated to v%s. Please restart vc.", latest), func() {
					a.closeDialog("error")
				})
			}
			a.ModalOpen = true
			a.TviewApp.SetFocus(a.Pages)
		})
	}()
}

type spinner struct {
	view *tview.TextView
	done chan struct{}
}

func (a *App) showSpinner(label string) *spinner {
	sv := tview.NewTextView()
	sv.SetBackgroundColor(theme.ColorDialogBg)
	sv.SetTextColor(theme.ColorDialogFg)
	sv.SetBorder(true)
	sv.SetBorderColor(theme.ColorDialogBorder)
	sv.SetTextAlign(tview.AlignCenter)

	if len(label) > 20 {
		label = label[:20] + "..."
	}
	boxW := len(label) + 6
	if boxW < 18 {
		boxW = 18
	}
	_, _, screenW, screenH := a.Pages.GetInnerRect()
	sv.SetRect(screenW-boxW-1, screenH-4, boxW, 3)
	sv.SetText(label + " |")

	a.Pages.AddPage("update-spinner", sv, false, true)
	a.focusActiveTable()

	done := make(chan struct{})
	go func() {
		spinChars := [4]rune{'|', '/', '-', '\\'}
		spinIdx := 0
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
					sv.SetText(fmt.Sprintf("%s %c", label, ch))
				})
			}
		}
	}()

	return &spinner{view: sv, done: done}
}

func (a *App) removeSpinner(s *spinner) {
	close(s.done)
	a.TviewApp.QueueUpdateDraw(func() {
		saved := a.activePanel
		a.Pages.RemovePage("update-spinner")
		a.activePanel = saved
		a.focusActiveTable()
		a.updatePanelStates()
	})
}

func fetchLatestRelease() (*ghRelease, error) {
	resp, err := http.Get("https://api.github.com/repos/feherk/vc/releases/latest")
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var rel ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&rel); err != nil {
		return nil, err
	}
	return &rel, nil
}

// isNewer returns true if latest is newer than current (semver: major.minor.patch).
func isNewer(current, latest string) bool {
	cur := parseSemver(current)
	lat := parseSemver(latest)
	for i := 0; i < 3; i++ {
		if lat[i] > cur[i] {
			return true
		}
		if lat[i] < cur[i] {
			return false
		}
	}
	return false
}

func parseSemver(v string) [3]int {
	v = strings.TrimPrefix(v, "v")
	parts := strings.SplitN(v, ".", 3)
	var result [3]int
	for i := 0; i < len(parts) && i < 3; i++ {
		result[i], _ = strconv.Atoi(parts[i])
	}
	return result
}

func downloadAndReplace(url string) error {
	exePath, err := os.Executable()
	if err != nil {
		return err
	}

	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("download returned %d", resp.StatusCode)
	}

	// Write to temp file in the same directory (for atomic rename on same filesystem)
	dir := filepath.Dir(exePath)
	tmp, err := os.CreateTemp(dir, "vc-update-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()

	_, err = io.Copy(tmp, resp.Body)
	tmp.Close()
	if err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Chmod(tmpPath, 0755); err != nil {
		os.Remove(tmpPath)
		return err
	}

	if err := os.Rename(tmpPath, exePath); err != nil {
		os.Remove(tmpPath)
		return err
	}

	return nil
}
