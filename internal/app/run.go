package app

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/feherkaroly/vc/internal/model"
)

// runCommand returns the command that executes the entry in the terminal,
// or nil when the entry is not something we run ourselves (documents,
// media, unknown types → system default application via OpenFile).
//
// Rules:
//   - Unix: shell scripts (.sh/.bash/.zsh/.command) always run — through the
//     interpreter when the exec bit is missing, directly (shebang) when it is set.
//     Other files run directly only with the exec bit AND a script-like or
//     missing extension; this keeps exFAT/SMB volumes (where every file is
//     0777) from "executing" images and documents.
//   - Windows: .bat/.cmd/.exe/.com via cmd /c, .ps1 via powershell.
func runCommand(path string, e *model.FileEntry) *exec.Cmd {
	ext := strings.ToLower(filepath.Ext(e.Name))
	if runtime.GOOS == "windows" {
		switch ext {
		case ".bat", ".cmd", ".exe", ".com":
			return exec.Command("cmd", "/c", path)
		case ".ps1":
			return exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", path)
		}
		return nil
	}

	executable := e.Mode&0111 != 0
	switch ext {
	case ".sh", ".bash", ".command":
		if executable {
			return exec.Command(path)
		}
		return exec.Command("bash", path)
	case ".zsh":
		if executable {
			return exec.Command(path)
		}
		return exec.Command("zsh", path)
	case "", ".py", ".pl", ".rb", ".run", ".bin":
		if executable {
			return exec.Command(path)
		}
	}
	return nil
}

// OpenOrRun is the Enter action on a file: scripts and executables run in
// the terminal (TUI suspended, like F4), everything else goes to OpenFile.
func (a *App) OpenOrRun() {
	p := a.GetActivePanel()
	e := p.CurrentEntry()
	if e == nil || e.IsDir || p.IsRemote() {
		a.OpenFile()
		return
	}
	path := filepath.Join(p.Path, e.Name)
	cmd := runCommand(path, e)
	if cmd == nil {
		a.OpenFile()
		return
	}

	a.TviewApp.Suspend(func() {
		// Ctrl+C must only stop the script, not vc.
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, os.Interrupt)
		defer signal.Stop(sig)

		cmd.Dir = p.Path
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		fmt.Printf("\033[2J\033[H» %s\n\n", e.Name)
		err := cmd.Run()
		fmt.Println()
		if err != nil {
			fmt.Printf("[%s] %v\n", e.Name, err)
		} else {
			fmt.Printf("[%s] finished.\n", e.Name)
		}
		fmt.Print("Press Enter to return to vc...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	})

	p.Refresh()
	a.GetInactivePanel().Refresh()
}
