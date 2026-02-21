# vc - Volkov Commander Clone

A dual-pane file manager for the terminal, a clone of the classic [Volkov Commander](https://en.wikipedia.org/wiki/Volkov_Commander) from the DOS era.

Built with Go, using [tview](https://github.com/rivo/tview) and [tcell](https://github.com/gdamore/tcell).

## Features

- Dual-pane navigation with Full and Brief display modes
- File operations: Copy (F5), Move/Rename (F6), Delete (F8), MkDir (F7)
- File viewer (F3) with zip archive content listing
- Zip compression (F2) for selected files/directories
- Open files with system default application (Enter)
- File editor integration via `$EDITOR` (F4)
- Quick search (Ctrl+S)
- Directory size calculation (Space)
- Multi-file selection (Insert/F11)
- Sorting by name, extension, size, or time
- SFTP/FTPS remote filesystem support (F1)
- Windows drive switching (Backspace at drive root)
- Shell command execution from built-in command line
- Classic DOS blue theme

## Installation

Download the latest release for your platform from the [Releases](https://github.com/feherk/vc/releases) page.

### Linux

```bash
mkdir -p ~/.local/bin
curl -L -o ~/.local/bin/vc https://github.com/feherk/vc/releases/latest/download/vc-linux-amd64
chmod +x ~/.local/bin/vc
```

### macOS (Apple Silicon)

```bash
mkdir -p ~/.local/bin
curl -L -o ~/.local/bin/vc https://github.com/feherk/vc/releases/latest/download/vc-darwin-arm64
chmod +x ~/.local/bin/vc
```

### macOS (Intel)

```bash
mkdir -p ~/.local/bin
curl -L -o ~/.local/bin/vc https://github.com/feherk/vc/releases/latest/download/vc-darwin-amd64
chmod +x ~/.local/bin/vc
```

Or download `VC-x.x.x-macOS.dmg` from [Releases](https://github.com/feherk/vc/releases) and drag VC to Applications.

### Windows (cmd)

```cmd
curl -L -o vc.exe https://github.com/feherk/vc/releases/latest/download/vc-windows-amd64.exe
```

> **Note:** Make sure `~/.local/bin` is in your `$PATH`. Add `export PATH="$HOME/.local/bin:$PATH"` to your `~/.bashrc` or `~/.zshrc` if needed.

### Build from source

```bash
go install github.com/feherkaroly/vc@latest
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Tab | Switch panel |
| Enter | Open file / Enter directory |
| Backspace | Go to parent directory / Change drive (Windows) |
| Insert/F11 | Toggle selection |
| Space | Calculate directory size |
| Ctrl+S | Quick search |
| Ctrl+R | Refresh both panels |
| F1 | Server connections (SFTP/FTPS) |
| F2 | Zip selected files |
| F3 | View file / View zip contents |
| F4 | Edit file ($EDITOR) |
| F5 | Copy |
| F6 | Move / Rename |
| F7 | Create directory |
| F8 | Delete |
| F9 | Menu |
| F10 | Quit |
| F12 | Copy filename to command line |

## License

MIT
