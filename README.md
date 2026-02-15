# vc - Volkov Commander

A dual-pane file manager for the terminal, inspired by the classic Volkov Commander.

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
- Multi-file selection (Insert)
- Sorting by name, extension, size, or time
- Shell command execution from built-in command line
- Classic DOS blue theme

## Installation

Download the latest release for your platform from the [Releases](https://github.com/feherk/vc/releases) page.

### Build from source

```bash
go install github.com/feherkaroly/vc@latest
```

## Keyboard Shortcuts

| Key | Action |
|-----|--------|
| Tab | Switch panel |
| Enter | Open file / Enter directory |
| Backspace | Go to parent directory |
| Insert | Toggle selection |
| Space | Calculate directory size |
| Ctrl+S | Quick search |
| Ctrl+R | Refresh both panels |
| F2 | Zip selected files |
| F3 | View file / View zip contents |
| F4 | Edit file ($EDITOR) |
| F5 | Copy |
| F6 | Move / Rename |
| F7 | Create directory |
| F8 | Delete |
| F9 | Menu |
| F10 | Quit |

## License

MIT
