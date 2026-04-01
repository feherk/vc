package dialog

import (
	"fmt"
	"os"
	"unicode/utf8"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// ChmodParams holds input parameters for the chmod dialog.
type ChmodParams struct {
	Name    string
	Mode    os.FileMode
	Owner   string
	Group   string
	OwnerID int
	GroupID int
	IsDir   bool
	IsLocal bool
	ACL     [3][3]bool
	HasACL  bool
	Index   int // current file index (0-based)
	Total   int // total file count
}

// ChmodResult holds the result of the chmod dialog.
type ChmodResult struct {
	Mode    os.FileMode
	Owner   string
	Group   string
	ACL     [3][3]bool
	Recurse bool
	Changed bool
}

// Sections:
//
//	0 = Owner perms [rwx]
//	1 = Group perms [rwx]
//	2 = Other perms [rwx]
//	3 = Special bits (SetUID, SetGID, Sticky)
//	4 = Recurse checkbox (directories only)
//	5 = Owner input
//	6 = Group input
//	7 = ACL (if available, one section with 9 positions)
//	last = Buttons
const (
	secPermOwner  = 0
	secPermGroup  = 1
	secPermOther  = 2
	secSpecial    = 3
	secRecurse    = 4
	secOwnerInput = 5
	secGroupInput = 6
)

type chmodBox struct {
	*tview.Box
	params   ChmodParams
	callback func(ChmodResult)
	onCancel func()

	perms    [3][3]bool
	special  [3]bool // [0]=SetUID, [1]=SetGID, [2]=Sticky
	recurse  bool
	ownerBuf string
	groupBuf string
	acl      [3][3]bool

	focusSection int
	focusCol     int
	buttonIdx    int // 0=OK, 1=Cancel

	// List picker overlay
	listActive   bool
	listItems    []string
	listFiltered []string
	listSelected int
	listOffset   int
	listTarget   int // secOwnerInput or secGroupInput
	listSearch   string
}

const listMaxVisible = 10

func newChmodBox(params ChmodParams, callback func(ChmodResult), onCancel func()) *chmodBox {
	b := &chmodBox{
		Box:      tview.NewBox(),
		params:   params,
		callback: callback,
		onCancel: onCancel,
		ownerBuf: params.Owner,
		groupBuf: params.Group,
		acl:      params.ACL,
	}

	mode := params.Mode.Perm()
	for i := 0; i < 3; i++ {
		shift := uint((2 - i) * 3)
		b.perms[i][0] = mode&(1<<(shift+2)) != 0
		b.perms[i][1] = mode&(1<<(shift+1)) != 0
		b.perms[i][2] = mode&(1<<shift) != 0
	}

	// Special bits
	b.special[0] = params.Mode&os.ModeSetuid != 0
	b.special[1] = params.Mode&os.ModeSetgid != 0
	b.special[2] = params.Mode&os.ModeSticky != 0

	// ALL key handling in SetInputCapture — this runs reliably
	// in tview's pipeline before InputHandler, for all keys.
	b.Box.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		b.handleKey(event)
		return nil // always consume
	})

	return b
}

func (b *chmodBox) buildMode() os.FileMode {
	var mode os.FileMode
	for i := 0; i < 3; i++ {
		shift := uint((2 - i) * 3)
		if b.perms[i][0] {
			mode |= 1 << (shift + 2)
		}
		if b.perms[i][1] {
			mode |= 1 << (shift + 1)
		}
		if b.perms[i][2] {
			mode |= 1 << shift
		}
	}
	if b.special[0] {
		mode |= os.ModeSetuid
	}
	if b.special[1] {
		mode |= os.ModeSetgid
	}
	if b.special[2] {
		mode |= os.ModeSticky
	}
	if b.params.IsDir {
		mode |= os.ModeDir
	}
	return mode
}

func (b *chmodBox) octalString() string {
	sp := 0
	if b.special[0] {
		sp |= 4
	}
	if b.special[1] {
		sp |= 2
	}
	if b.special[2] {
		sp |= 1
	}
	if sp > 0 {
		return fmt.Sprintf("%d%03o", sp, b.buildMode().Perm())
	}
	return fmt.Sprintf("%03o", b.buildMode().Perm())
}

func (b *chmodBox) showRecurse() bool {
	return b.params.IsDir
}

func (b *chmodBox) showACL() bool {
	return b.params.HasACL && b.params.IsDir && b.params.IsLocal
}

func (b *chmodBox) ownerSection() int {
	if b.showRecurse() {
		return secRecurse + 1
	}
	return secRecurse
}

func (b *chmodBox) groupSection() int {
	return b.ownerSection() + 1
}

func (b *chmodBox) aclSection() int {
	return b.groupSection() + 1
}

func (b *chmodBox) buttonsSection() int {
	if b.showACL() {
		return b.aclSection() + 1
	}
	return b.groupSection() + 1
}

func (b *chmodBox) maxSections() int {
	return b.buttonsSection() + 1
}

// --- Draw ---

func (b *chmodBox) Draw(screen tcell.Screen) {
	x, y, width, totalH := b.GetRect()

	boxW := 54
	boxH := 11 // extra row for special bits
	if b.showRecurse() {
		boxH++
	}
	if b.showACL() {
		boxH += 3
	}

	bx := x + (width-boxW)/2
	by := y + (totalH-boxH)/2

	bg := theme.ColorDialogBg
	fg := theme.ColorDialogFg
	brd := theme.ColorDialogBorder

	// Fill background
	for r := 0; r < boxH; r++ {
		for c := 0; c < boxW; c++ {
			screen.SetContent(bx+c, by+r, ' ', nil,
				tcell.StyleDefault.Foreground(fg).Background(bg))
		}
	}
	chmodDrawBorder(screen, bx, by, boxW, boxH, brd, bg)

	// Title
	title := " Chmod-Chown "
	chmodDrawStr(screen, bx+(boxW-runeLen(title))/2, by, title, theme.ColorHeaderFg, bg)

	// Filename + counter
	row := by + 1
	name := truncateRunes(b.params.Name, boxW-12)
	chmodDrawStr(screen, bx+2, row, name, fg, bg)
	if b.params.Total > 1 {
		cnt := fmt.Sprintf("(%d/%d)", b.params.Index+1, b.params.Total)
		chmodDrawStr(screen, bx+boxW-2-runeLen(cnt), row, cnt, fg, bg)
	}

	// Separator
	row++
	drawHSep(screen, bx, row, boxW, brd, bg)

	// Headers
	row++
	chmodDrawStr(screen, bx+3, row, "Owner", tcell.ColorYellow, bg)
	chmodDrawStr(screen, bx+11, row, "Group", tcell.ColorYellow, bg)
	chmodDrawStr(screen, bx+21, row, "Other", tcell.ColorYellow, bg)
	chmodDrawStr(screen, bx+30, row, "Owner", tcell.ColorYellow, bg)
	chmodDrawStr(screen, bx+42, row, "Group", tcell.ColorYellow, bg)

	// Perm bits + owner/group fields
	row++
	b.drawPermBits(screen, bx+2, row, fg, bg)
	b.drawOwnerGroup(screen, bx+28, row, fg, bg)

	// Special bits
	row++
	b.drawSpecialBits(screen, bx+2, row, fg, bg)

	// Recurse checkbox (directories only)
	if b.showRecurse() {
		row++
		b.drawRecurse(screen, bx+2, row, fg, bg)
	}

	// Octal
	row++
	chmodDrawStr(screen, bx+2, row, fmt.Sprintf("Octal: %s", b.octalString()), fg, bg)

	// Separator
	row++
	drawHSep(screen, bx, row, boxW, brd, bg)

	if b.showACL() {
		row++
		chmodDrawStr(screen, bx+2, row, "Default ACL:", tcell.ColorYellow, bg)
		row++
		b.drawACLBits(screen, bx+2, row, fg, bg)
		row++
		drawHSep(screen, bx, row, boxW, brd, bg)
	}

	// Buttons
	row++
	b.drawButtons(screen, bx, row, boxW, bg)

	// List overlay (on top of everything)
	if b.listActive {
		b.drawList(screen, bx, by, boxW, boxH)
	}
}

func (b *chmodBox) drawPermBits(screen tcell.Screen, x, y int, fg, bg tcell.Color) {
	cx := x
	for grp := 0; grp < 3; grp++ {
		chmodDrawStr(screen, cx, y, "[", fg, bg)
		cx++
		chars := [3]rune{'r', 'w', 'x'}
		for bit := 0; bit < 3; bit++ {
			display := '-'
			if b.perms[grp][bit] {
				display = chars[bit]
			}
			bitFg, bitBg := fg, bg
			if b.focusSection == grp && b.focusCol == bit {
				bitFg = theme.ColorButtonFg
				bitBg = theme.ColorButtonBg
			}
			screen.SetContent(cx, y, display, nil,
				tcell.StyleDefault.Foreground(bitFg).Background(bitBg))
			cx++
		}
		chmodDrawStr(screen, cx, y, "]", fg, bg)
		cx += 3
	}
}

func (b *chmodBox) drawSpecialBits(screen tcell.Screen, x, y int, fg, bg tcell.Color) {
	labels := [3]string{"SetUID", "SetGID", "Sticky"}
	cx := x
	for i := 0; i < 3; i++ {
		ch := ' '
		if b.special[i] {
			ch = 'x'
		}
		// Draw checkbox
		chmodDrawStr(screen, cx, y, "[", fg, bg)
		chFg, chBg := fg, bg
		if b.focusSection == secSpecial && b.focusCol == i {
			chFg = theme.ColorButtonFg
			chBg = theme.ColorButtonBg
		}
		screen.SetContent(cx+1, y, ch, nil,
			tcell.StyleDefault.Foreground(chFg).Background(chBg))
		chmodDrawStr(screen, cx+2, y, "] ", fg, bg)
		chmodDrawStr(screen, cx+4, y, labels[i], fg, bg)
		cx += 4 + runeLen(labels[i]) + 2
	}
}

func (b *chmodBox) drawOwnerGroup(screen tcell.Screen, x, y int, fg, bg tcell.Color) {
	// Owner
	oFg, oBg := fg, theme.ColorPanelBg
	if b.focusSection == b.ownerSection() {
		oFg, oBg = theme.ColorButtonFg, theme.ColorButtonBg
	}
	chmodDrawStr(screen, x, y, "[", fg, bg)
	chmodDrawStr(screen, x+1, y, padRight(truncateRunes(b.ownerBuf, 10), 10), oFg, oBg)
	chmodDrawStr(screen, x+11, y, "]", fg, bg)

	// Group
	gFg, gBg := fg, theme.ColorPanelBg
	if b.focusSection == b.groupSection() {
		gFg, gBg = theme.ColorButtonFg, theme.ColorButtonBg
	}
	chmodDrawStr(screen, x+13, y, "[", fg, bg)
	chmodDrawStr(screen, x+14, y, padRight(truncateRunes(b.groupBuf, 10), 10), gFg, gBg)
	chmodDrawStr(screen, x+24, y, "]", fg, bg)
}

func (b *chmodBox) drawRecurse(screen tcell.Screen, x, y int, fg, bg tcell.Color) {
	ch := ' '
	if b.recurse {
		ch = 'x'
	}
	chmodDrawStr(screen, x, y, "[", fg, bg)
	chFg, chBg := fg, bg
	if b.focusSection == secRecurse {
		chFg = theme.ColorButtonFg
		chBg = theme.ColorButtonBg
	}
	screen.SetContent(x+1, y, ch, nil,
		tcell.StyleDefault.Foreground(chFg).Background(chBg))
	chmodDrawStr(screen, x+2, y, "] Recurse", fg, bg)
}

func (b *chmodBox) drawACLBits(screen tcell.Screen, x, y int, fg, bg tcell.Color) {
	labels := [3]string{"Owner", "Group", "Other"}
	cx := x
	for grp := 0; grp < 3; grp++ {
		lbl := labels[grp] + " ["
		chmodDrawStr(screen, cx, y, lbl, fg, bg)
		cx += runeLen(lbl)
		chars := [3]rune{'r', 'w', 'x'}
		for bit := 0; bit < 3; bit++ {
			display := '-'
			if b.acl[grp][bit] {
				display = chars[bit]
			}
			bitFg, bitBg := fg, bg
			if b.focusSection == b.aclSection() && b.focusCol == grp*3+bit {
				bitFg = theme.ColorButtonFg
				bitBg = theme.ColorButtonBg
			}
			screen.SetContent(cx, y, display, nil,
				tcell.StyleDefault.Foreground(bitFg).Background(bitBg))
			cx++
		}
		chmodDrawStr(screen, cx, y, "]", fg, bg)
		cx += 2
	}
}

func (b *chmodBox) drawButtons(screen tcell.Screen, bx, y, boxW int, bg tcell.Color) {
	btn := b.buttonsSection()
	okLbl := " OK "
	caLbl := " Cancel "

	okFg, okBg := theme.ColorButtonFg, theme.ColorButtonBg
	caFg, caBg := theme.ColorButtonFg, theme.ColorButtonBg
	if b.focusSection == btn && b.buttonIdx == 0 {
		okFg, okBg = theme.ColorDialogBg, tcell.ColorWhite
	}
	if b.focusSection == btn && b.buttonIdx == 1 {
		caFg, caBg = theme.ColorDialogBg, tcell.ColorWhite
	}

	totalW := runeLen(okLbl) + 4 + runeLen(caLbl)
	sx := bx + (boxW-totalW)/2
	chmodDrawStr(screen, sx, y, okLbl, okFg, okBg)
	chmodDrawStr(screen, sx+runeLen(okLbl)+4, y, caLbl, caFg, caBg)
}

func (b *chmodBox) drawList(screen tcell.Screen, bx, by, boxW, boxH int) {
	items := b.listFiltered
	if len(items) == 0 {
		return
	}

	visible := len(items)
	if visible > listMaxVisible {
		visible = listMaxVisible
	}

	listW := 24
	listH := visible + 2 // border
	// Position: next to the owner/group field
	lx := bx + 28
	if b.listTarget == b.groupSection() {
		lx = bx + 41
	}
	ly := by + 4 // below the field row

	// Ensure list fits on screen
	_, _, screenW, screenH := b.Box.GetRect()
	if lx+listW > bx+screenW {
		lx = bx + boxW - listW - 1
	}
	if ly+listH > by+screenH {
		ly = by + boxH - listH - 1
	}

	bg := theme.ColorPanelBg
	fg := theme.ColorDialogFg
	brd := theme.ColorDialogBorder

	// Fill + border
	for r := 0; r < listH; r++ {
		for c := 0; c < listW; c++ {
			screen.SetContent(lx+c, ly+r, ' ', nil,
				tcell.StyleDefault.Foreground(fg).Background(bg))
		}
	}
	chmodDrawBorder(screen, lx, ly, listW, listH, brd, bg)

	// Search indicator in title
	titleStr := ""
	if b.listSearch != "" {
		titleStr = " " + b.listSearch + " "
	}
	if titleStr != "" {
		chmodDrawStr(screen, lx+(listW-runeLen(titleStr))/2, ly, titleStr, tcell.ColorYellow, bg)
	}

	// Items
	for i := 0; i < visible; i++ {
		idx := b.listOffset + i
		if idx >= len(items) {
			break
		}
		itemFg, itemBg := fg, bg
		if idx == b.listSelected {
			itemFg = theme.ColorCursorFg
			itemBg = theme.ColorCursorBg
		}
		// Clear row
		for c := 1; c < listW-1; c++ {
			screen.SetContent(lx+c, ly+1+i, ' ', nil,
				tcell.StyleDefault.Foreground(itemFg).Background(itemBg))
		}
		name := truncateRunes(items[idx], listW-4)
		chmodDrawStr(screen, lx+2, ly+1+i, name, itemFg, itemBg)
	}
}

// --- Input handling ---

// handleKey processes ALL keyboard input for both the main dialog and the list picker.
func (b *chmodBox) handleKey(event *tcell.EventKey) {
	if b.listActive {
		b.handleListKey(event)
		return
	}

	switch event.Key() {
	case tcell.KeyEscape:
		b.onCancel()
	case tcell.KeyTab:
		b.focusSection = (b.focusSection + 1) % b.maxSections()
		b.focusCol = 0
		b.buttonIdx = 0
	case tcell.KeyBacktab:
		b.focusSection--
		if b.focusSection < 0 {
			b.focusSection = b.maxSections() - 1
		}
		b.focusCol = 0
		b.buttonIdx = 0
	case tcell.KeyLeft:
		b.moveLeft()
	case tcell.KeyRight:
		b.moveRight()
	case tcell.KeyEnter:
		switch {
		case b.focusSection == b.ownerSection():
			b.openList(b.ownerSection())
		case b.focusSection == b.groupSection():
			b.openList(b.groupSection())
		case b.focusSection == b.buttonsSection():
			if b.buttonIdx == 0 {
				b.apply()
			} else {
				b.onCancel()
			}
		}
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if b.focusSection == b.ownerSection() && len(b.ownerBuf) > 0 {
			b.ownerBuf = removeLastRune(b.ownerBuf)
		} else if b.focusSection == b.groupSection() && len(b.groupBuf) > 0 {
			b.groupBuf = removeLastRune(b.groupBuf)
		}
	case tcell.KeyRune:
		ch := event.Rune()
		if b.focusSection == b.ownerSection() {
			if runeLen(b.ownerBuf) < 32 {
				b.ownerBuf += string(ch)
			}
		} else if b.focusSection == b.groupSection() {
			if runeLen(b.groupBuf) < 32 {
				b.groupBuf += string(ch)
			}
		} else if ch == ' ' {
			b.toggleCurrent()
		} else if b.focusSection >= secPermOwner && b.focusSection <= secPermOther {
			b.togglePermByKey(ch, b.focusSection)
		} else if b.focusSection == secSpecial {
			// no letter-toggle for special bits, Space only
		} else if b.showACL() && b.focusSection == b.aclSection() {
			b.toggleACLByKey(ch, b.focusCol/3)
		}
	}
}

func (b *chmodBox) InputHandler() func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	return b.WrapInputHandler(func(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
		// All handling is done in SetInputCapture
	})
}

func (b *chmodBox) togglePermByKey(ch rune, grp int) {
	switch ch {
	case 'r', 'R':
		b.perms[grp][0] = !b.perms[grp][0]
	case 'w', 'W':
		b.perms[grp][1] = !b.perms[grp][1]
	case 'x', 'X':
		b.perms[grp][2] = !b.perms[grp][2]
	}
}

func (b *chmodBox) toggleACLByKey(ch rune, grp int) {
	if grp < 0 || grp > 2 {
		return
	}
	switch ch {
	case 'r', 'R':
		b.acl[grp][0] = !b.acl[grp][0]
	case 'w', 'W':
		b.acl[grp][1] = !b.acl[grp][1]
	case 'x', 'X':
		b.acl[grp][2] = !b.acl[grp][2]
	}
}

func (b *chmodBox) moveLeft() {
	switch {
	case b.focusSection >= secPermOwner && b.focusSection <= secPermOther:
		if b.focusCol > 0 {
			b.focusCol--
		}
	case b.focusSection == secSpecial:
		if b.focusCol > 0 {
			b.focusCol--
		}
	case b.showACL() && b.focusSection == b.aclSection():
		if b.focusCol > 0 {
			b.focusCol--
		}
	case b.focusSection == b.buttonsSection():
		if b.buttonIdx > 0 {
			b.buttonIdx--
		}
	}
}

func (b *chmodBox) moveRight() {
	switch {
	case b.focusSection >= secPermOwner && b.focusSection <= secPermOther:
		if b.focusCol < 2 {
			b.focusCol++
		}
	case b.focusSection == secSpecial:
		if b.focusCol < 2 {
			b.focusCol++
		}
	case b.showACL() && b.focusSection == b.aclSection():
		if b.focusCol < 8 {
			b.focusCol++
		}
	case b.focusSection == b.buttonsSection():
		if b.buttonIdx < 1 {
			b.buttonIdx++
		}
	}
}

func (b *chmodBox) toggleCurrent() {
	switch {
	case b.focusSection >= secPermOwner && b.focusSection <= secPermOther:
		b.perms[b.focusSection][b.focusCol] = !b.perms[b.focusSection][b.focusCol]
	case b.focusSection == secSpecial:
		b.special[b.focusCol] = !b.special[b.focusCol]
	case b.focusSection == secRecurse && b.showRecurse():
		b.recurse = !b.recurse
	case b.showACL() && b.focusSection == b.aclSection():
		grp := b.focusCol / 3
		bit := b.focusCol % 3
		b.acl[grp][bit] = !b.acl[grp][bit]
	}
}

func (b *chmodBox) apply() {
	newMode := b.buildMode()
	changed := newMode != b.params.Mode ||
		b.ownerBuf != b.params.Owner ||
		b.groupBuf != b.params.Group ||
		b.acl != b.params.ACL ||
		b.recurse
	b.callback(ChmodResult{
		Mode:    newMode,
		Owner:   b.ownerBuf,
		Group:   b.groupBuf,
		ACL:     b.acl,
		Recurse: b.recurse,
		Changed: changed,
	})
}

// --- List picker ---

func (b *chmodBox) openList(target int) {
	var items []string
	if target == b.ownerSection() {
		items = ListUsers()
	} else {
		items = ListGroups()
	}
	if len(items) == 0 {
		return
	}
	b.listActive = true
	b.listItems = items
	b.listFiltered = items
	b.listSelected = 0
	b.listOffset = 0
	b.listTarget = target
	b.listSearch = ""

	// Try to select current value
	current := b.ownerBuf
	if target == b.groupSection() {
		current = b.groupBuf
	}
	for i, item := range items {
		if item == current {
			b.listSelected = i
			if i >= listMaxVisible {
				b.listOffset = i - listMaxVisible/2
			}
			break
		}
	}
}

func (b *chmodBox) handleListKey(event *tcell.EventKey) {
	switch event.Key() {
	case tcell.KeyEscape:
		b.listActive = false
		return
	case tcell.KeyEnter:
		if len(b.listFiltered) > 0 && b.listSelected < len(b.listFiltered) {
			val := b.listFiltered[b.listSelected]
			if b.listTarget == b.ownerSection() {
				b.ownerBuf = val
			} else {
				b.groupBuf = val
			}
		}
		b.listActive = false
		return
	case tcell.KeyDown:
		if b.listSelected < len(b.listFiltered)-1 {
			b.listSelected++
			if b.listSelected >= b.listOffset+listMaxVisible {
				b.listOffset++
			}
		}
		return
	case tcell.KeyUp:
		if b.listSelected > 0 {
			b.listSelected--
			if b.listSelected < b.listOffset {
				b.listOffset = b.listSelected
			}
		}
		return
	case tcell.KeyBackspace, tcell.KeyBackspace2:
		if len(b.listSearch) > 0 {
			b.listSearch = removeLastRune(b.listSearch)
			b.filterList()
		}
		return
	case tcell.KeyRune:
		b.listSearch += string(event.Rune())
		b.filterList()
		return
	}
}

func (b *chmodBox) filterList() {
	if b.listSearch == "" {
		b.listFiltered = b.listItems
	} else {
		b.listFiltered = nil
		for _, item := range b.listItems {
			if len(item) >= len(b.listSearch) && containsLower(item, b.listSearch) {
				b.listFiltered = append(b.listFiltered, item)
			}
		}
	}
	b.listSelected = 0
	b.listOffset = 0
}

func containsLower(s, sub string) bool {
	sl := make([]rune, 0, len(s))
	for _, r := range s {
		if r >= 'A' && r <= 'Z' {
			r += 32
		}
		sl = append(sl, r)
	}
	subl := make([]rune, 0, len(sub))
	for _, r := range sub {
		if r >= 'A' && r <= 'Z' {
			r += 32
		}
		subl = append(subl, r)
	}
	ss := string(sl)
	ssub := string(subl)
	for i := 0; i <= len(ss)-len(ssub); i++ {
		if ss[i:i+len(ssub)] == ssub {
			return true
		}
	}
	return false
}

// --- Mouse ---

func (b *chmodBox) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return b.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		if action != tview.MouseLeftClick {
			return false, nil
		}

		if b.listActive {
			// Close list on any click
			b.listActive = false
			return true, nil
		}

		mx, my := event.Position()
		x, y, width, totalH := b.GetRect()

		boxW := 54
		boxH := 11
		if b.showRecurse() {
			boxH++
		}
		if b.showACL() {
			boxH += 3
		}
		bx := x + (width-boxW)/2
		by := y + (totalH-boxH)/2

		if mx < bx || mx >= bx+boxW || my < by || my >= by+boxH {
			b.onCancel()
			return true, nil
		}

		// Buttons row
		btnRow := by + boxH - 2
		if my == btnRow {
			okLbl := " OK "
			caLbl := " Cancel "
			totalW := runeLen(okLbl) + 4 + runeLen(caLbl)
			sx := bx + (boxW-totalW)/2
			if mx >= sx && mx < sx+runeLen(okLbl) {
				b.apply()
				return true, nil
			}
			caStart := sx + runeLen(okLbl) + 4
			if mx >= caStart && mx < caStart+runeLen(caLbl) {
				b.onCancel()
				return true, nil
			}
		}

		return true, nil
	})
}

// ShowChmodDialog displays the chmod dialog.
func ShowChmodDialog(pages *tview.Pages, params ChmodParams, callback func(ChmodResult), onCancel func()) {
	fb := newChmodBox(params, callback, onCancel)
	pages.AddPage("chmod", fb, true, true)
}

// --- Drawing helpers ---

func chmodDrawBorder(screen tcell.Screen, x, y, w, h int, fg, bg tcell.Color) {
	st := tcell.StyleDefault.Foreground(fg).Background(bg)
	screen.SetContent(x, y, '┌', nil, st)
	screen.SetContent(x+w-1, y, '┐', nil, st)
	screen.SetContent(x, y+h-1, '└', nil, st)
	screen.SetContent(x+w-1, y+h-1, '┘', nil, st)
	for c := 1; c < w-1; c++ {
		screen.SetContent(x+c, y, '─', nil, st)
		screen.SetContent(x+c, y+h-1, '─', nil, st)
	}
	for r := 1; r < h-1; r++ {
		screen.SetContent(x, y+r, '│', nil, st)
		screen.SetContent(x+w-1, y+r, '│', nil, st)
	}
}

func drawHSep(screen tcell.Screen, x, y, w int, fg, bg tcell.Color) {
	st := tcell.StyleDefault.Foreground(fg).Background(bg)
	screen.SetContent(x, y, '├', nil, st)
	screen.SetContent(x+w-1, y, '┤', nil, st)
	for c := 1; c < w-1; c++ {
		screen.SetContent(x+c, y, '─', nil, st)
	}
}

func chmodDrawStr(screen tcell.Screen, x, y int, s string, fg, bg tcell.Color) {
	st := tcell.StyleDefault.Foreground(fg).Background(bg)
	col := 0
	for _, ch := range s {
		screen.SetContent(x+col, y, ch, nil, st)
		col++
	}
}

func runeLen(s string) int {
	return utf8.RuneCountInString(s)
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	runes := []rune(s)
	if max > 3 {
		return string(runes[:max-3]) + "..."
	}
	return string(runes[:max])
}

func padRight(s string, width int) string {
	n := utf8.RuneCountInString(s)
	if n >= width {
		return s
	}
	buf := make([]byte, 0, len(s)+width-n)
	buf = append(buf, s...)
	for i := 0; i < width-n; i++ {
		buf = append(buf, ' ')
	}
	return string(buf)
}

func removeLastRune(s string) string {
	if s == "" {
		return s
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}
