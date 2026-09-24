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
- Inline search — just start typing to jump to matching files
- Directory size calculation (Space)
- Multi-file selection (Insert/Ctrl+S)
- Sorting by name, extension, size, or time
- SFTP/FTPS remote filesystem support (F1) — copy/move/delete/rename/chmod and encrypt/decrypt work on remote panels too (compress/extract are local-only)
- SFTP via the system `ssh` when needed: hardware security keys (YubiKey / `sk-` keys), passphrase-protected keys, `~/.ssh/config` host aliases and the ssh-agent all work (see below)
- Windows drive switching (Backspace at drive root)
- Symlink display with `@` prefix and link target in footer
- File attributes dialog: chmod/chown with searchable owner/group picker
- Default ACL support for directories on Linux (`sudo apt install acl`)
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

The macOS binary is signed with a Developer ID and notarized by Apple, so it runs without Gatekeeper warnings. Intel Macs are not supported.

### Windows (cmd)

```cmd
curl -L -o vc.exe https://github.com/feherk/vc/releases/latest/download/vc-windows-amd64.exe
```

> **Note:** Make sure `~/.local/bin` is in your `$PATH`. Add `export PATH="$HOME/.local/bin:$PATH"` to your `~/.bashrc` or `~/.zshrc` if needed.

### SFTP with a YubiKey, ssh-agent or `~/.ssh/config`

The built-in SSH client handles plain private keys and passwords. In every other case vc runs the system `ssh` (on macOS the Homebrew OpenSSH, since Apple's build has no security key support) and speaks SFTP through it:

- **Key Path** points to a hardware-backed key (`id_yk`, `id_ed25519_sk`) or a passphrase-protected key — a bare file name is looked up in `~/.ssh/`, `~/` is expanded
- the **System ssh** checkbox is ticked in the server dialog
- both Password and Key Path are empty — then Host may be a `~/.ssh/config` alias, and User/Port can stay empty too

While connecting, vc leaves the terminal to ssh so PIN, passphrase and touch prompts are visible; Ctrl+C cancels the connection only.

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
| Insert/Ctrl+S | Toggle selection |
| Space | Calculate directory size |
| Type letters | Inline search — jump to matching file/directory |
| Escape | Cancel inline search |
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

## License

MIT
