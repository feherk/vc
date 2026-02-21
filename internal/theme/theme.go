package theme

import "github.com/gdamore/tcell/v2"

// DOS blue theme colors matching Volkov Commander.
var (
	// Panel
	ColorPanelBg      = tcell.NewRGBColor(0, 0, 128) // Navy
	ColorPanelBorder  = tcell.NewRGBColor(0, 170, 170) // Teal
	ColorNormalFile   = tcell.NewRGBColor(0, 170, 170) // Teal
	ColorDirectory    = tcell.ColorWhite
	ColorCursorFg     = tcell.ColorBlack
	ColorCursorBg     = tcell.NewRGBColor(0, 170, 170) // Teal
	ColorSelected     = tcell.ColorYellow
	ColorExecutable   = tcell.ColorGreen
	ColorArchive      = tcell.NewRGBColor(170, 0, 0)   // Dark red
	ColorDocument     = tcell.NewRGBColor(170, 0, 170)  // Magenta
	ColorMedia        = tcell.NewRGBColor(170, 85, 255) // Bright purple
	ColorSource       = tcell.NewRGBColor(85, 255, 255) // Bright cyan
	ColorEncrypted    = tcell.NewRGBColor(255, 85, 85)  // Bright red
	ColorSymlink      = tcell.NewRGBColor(85, 255, 255) // Bright cyan

	// Menu bar
	ColorMenuBarFg = tcell.ColorBlack
	ColorMenuBarBg = tcell.NewRGBColor(0, 170, 170)

	// Function key bar
	ColorFnKeyFg   = tcell.ColorWhite
	ColorFnKeyBg   = tcell.ColorBlack
	ColorFnLabelFg = tcell.ColorBlack
	ColorFnLabelBg = tcell.NewRGBColor(0, 170, 170)

	// Command line
	ColorCmdLineFg = tcell.ColorWhite
	ColorCmdLineBg = tcell.ColorBlack

	// Panel header/footer
	ColorHeaderFg = tcell.ColorWhite
	ColorHeaderBg = ColorPanelBg

	// Dialog
	ColorDialogFg     = tcell.ColorWhite
	ColorDialogBg     = tcell.NewRGBColor(0, 128, 128)
	ColorDialogBorder = tcell.ColorWhite
	ColorButtonFg     = tcell.ColorBlack
	ColorButtonBg     = tcell.NewRGBColor(0, 170, 170)

	// Inactive/Active panel border
	ColorInactiveBorder = tcell.NewRGBColor(0, 170, 170)
	ColorActiveBorder   = tcell.ColorWhite
)
