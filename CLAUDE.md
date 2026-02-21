# VC — Volkov Commander Clone

Dual-pane terminal file manager written in Go + tview/tcell, classic DOS blue theme.

- Repo: `github.com/feherk/vc` (public)
- Module: `github.com/feherkaroly/vc`

## Key Files

- `main.go` — entry point, version string, flag parsing
- `internal/app/app.go` — main App struct, file ops, dialogs, menu, config save/load
- `internal/app/update.go` — built-in self-update from GitHub releases
- `internal/app/keybindings.go` — all keyboard shortcuts + mouse handlers
- `internal/config/config.go` — Config struct (panels + servers), Load/Save to `~/.config/vc/config.json`
- `internal/menu/items.go` — MenuDefs struct, menu item builders
- `internal/menu/dropdown.go` — custom Dropdown widget (use this, NOT tview.List for menus)
- `internal/menu/menubar.go` — MenuBar widget with mouse click support
- `internal/dialog/server.go` — server list dialog (F1, uses tview.Table NOT List)
- `internal/dialog/format.go` — format selection dialog (custom formatBox widget)
- `internal/dialog/quickpaths.go` — Quick Paths dialog
- `internal/dialog/password.go` — password dialog with optional confirm mode
- `internal/fnbar/fnbar.go` — function key bar at bottom, clickable
- `internal/panel/render.go` — panel rendering (Brief/Full modes)
- `internal/theme/theme.go` — all color definitions
- `scripts/build-dmg.sh` — macOS DMG build with code signing + notarization

## Build & Release

- DMG: `./scripts/build-dmg.sh [version]` — builds universal binary, signs, notarizes, staples
- Darwin: `GOOS=darwin GOARCH=arm64/amd64 go build -ldflags "-s -w -X main.Version=X.Y.Z" -o dist/vc-darwin-arm64 .`
- Linux: `GOOS=linux GOARCH=amd64 go build -ldflags "-s -w -X main.Version=X.Y.Z" -o dist/vc-linux-amd64 .`
- Windows: `GOOS=windows GOARCH=amd64 go build -ldflags "-s -w -X main.Version=X.Y.Z" -o dist/vc-windows-amd64.exe .`
- Release assets: DMG + darwin arm64/amd64 + linux amd64 + windows amd64 (5 files)
- Release: `gh release create` on `feherk/vc`
- Version in `main.go` var, NOT ldflags (ldflags used in build scripts)

## Code Signing

- Identity: `Developer ID Application: Károly Fehér (YG66KQ8KDT)`
- Notarize profile: `vc-notarize` (stored in Keychain)

## Important Patterns & Lessons

### tview/tcell Gotchas

- **`tview.List` has broken background colors** — always prefer `tview.Table` or custom widgets
- **`tview.Table` inside dialog overlays doesn't render cells with `EnableMouse(true)`** — use custom Draw widgets (formatBox, Dropdown) that render directly to screen
- **Custom dialog widgets must NOT call `DrawForSubclass()`** — that fills the full-screen overlay with black. Only draw the dialog box area directly.
- **Table doesn't handle `MouseLeftDoubleClick`/`MouseRightClick`** → `consumed=false` → tview skips `draw()`. Fix: `go QueueUpdateDraw(func(){})` after every custom mouse action handler
- **tview generates `MouseLeftDoubleClick` INSTEAD of `MouseLeftClick` for the 2nd click** (not both) — need dual detection: native + custom 800ms backup
- **`SetRoot(pages, true)` triggers focus cascade** → must save/restore `activePanel` around it in `Run()`

### Mouse Support (v2.6.0)

- `EnableMouse(true)` in `New()` — activates tcell mouse reporting
- Left click: positions cursor (tview Table handles natively via `SetSelectable`)
- Double-click: `MouseLeftDoubleClick` + custom 800ms backup on `MouseLeftClick` → calls `p.Enter()`
- Right-click: `MouseRightClick` → `ToggleSelectionAt(idx)` from mouse Y position
- FnBar: `OnClick func(fn int)` + `MouseHandler()` — X slot calculation from click position
- MenuBar: `OnClick func(idx int)` + `MouseHandler()` — X position to menu item index
- Dropdown: `MouseHandler()` — click item to execute action, click outside returns not-consumed
- App-level `SetMouseCapture` dismisses menu on click outside (checks MenuBar y==0 and dropdown rect)
- `activeDropdown *menu.Dropdown` in App — tracks current dropdown for hit-testing
- FnBar/MenuBar OnClick handlers need `go QueueUpdateDraw(func(){})` to ensure dialogs are redrawn

### Application Logic

- `SaveConfig()` must preserve servers — calls `config.Load()` first then `saveConfigWithServers()`
- Menu dropdowns use custom `menu.Dropdown` widget, NOT `tview.List`
- Server dialog uses `tview.Table` NOT `tview.List`
- `CmdLineFocused` flag must be reset on: Tab (switchPanel), Backspace on empty input, Escape
- Panel focus tracking (`SetFocusFunc`) must ignore changes during `MenuActive`/`ModalOpen`
- F2 format dialog: context-dependent (encrypt for files + archives, decrypt for .enc, extract for archives, compress for dirs/multi)
- `runWithSpinner()` reusable helper for async ops with spinner
- Config stores `active_panel` (0=left, 1=right), restored on startup
- Quick Paths: config `QuickPaths map[string]string`, Alt+1..9 keybindings
- Recursive `showDialog` pattern (ServerDialog & QuickPathsDialog) for dialogs that reopen after sub-actions
- Self-update: Commands → Check for Updates, GitHub API (`feherk/vc/releases/latest`), asset pattern `vc-{GOOS}-{GOARCH}`, atomic binary replace via temp file + `os.Rename`
- Symlink: File menu → Symlink, creates symlinks in inactive panel dir pointing to active panel entries. Single entry → input dialog for link name, multiple → original names. Local-only (`os.Symlink`), no spinner needed.

### File Type Color Coding

- `fileColor()` in `render.go` — colors by extension after dir/executable checks
- Archives (.zip, .tar, .gz, etc.) → `ColorArchive` (dark red)
- Documents (.pdf, .doc, .xls, etc.) → `ColorDocument` (magenta)
- Media (.jpg, .mp3, .mp4, etc.) → `ColorMedia` (bright purple)
- Source/config (.go, .json, .yaml, etc.) → `ColorSource` (bright cyan)
- Encrypted (.enc) → `ColorEncrypted` (bright red)
- Color constants defined in `theme.go`

### Encryption

- AES-256-GCM + Argon2id
- File format: `[uint16 name len][name][salt][nonce][ciphertext+tag]`
- Filename header (length + name) is passed as AAD to `gcm.Seal`/`gcm.Open` — tampering with the stored filename is detected during decryption
- Argon2id params: time=1, memory=64MB, threads=4, keyLen=32

## User Preferences

- Language: Hungarian communication
- Workflow: build all platforms → commit → push → create/update GitHub release
- Prefers compact dialogs, clean theme, no unnecessary info shown
