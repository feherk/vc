package dialog

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// ShowFormatDialog displays a format selection dialog for compression.
// If singleFile is true and isEnc is false, an "encrypt" option is added.
// If singleFile is true and isEnc is true, a "decrypt" option is added.
func ShowFormatDialog(pages *tview.Pages, singleFile bool, isEnc bool, callback func(format string), onCancel func()) {
	formats := []string{"zip", "tar", "tar.gz"}
	if singleFile && isEnc {
		formats = append(formats, "decrypt")
	} else if singleFile && !isEnc {
		formats = append(formats, "encrypt")
	}

	table := tview.NewTable()
	table.SetBackgroundColor(theme.ColorDialogBg)
	table.SetSelectable(true, false)
	table.SetSelectedStyle(tcell.StyleDefault.
		Foreground(tcell.ColorBlack).
		Background(tcell.NewRGBColor(0, 170, 170)))

	for i, f := range formats {
		cell := tview.NewTableCell(" " + f).
			SetTextColor(theme.ColorDialogFg).
			SetBackgroundColor(theme.ColorDialogBg).
			SetExpansion(1)
		table.SetCell(i, 0, cell)
	}

	table.SetSelectedFunc(func(row, col int) {
		if row >= 0 && row < len(formats) {
			callback(formats[row])
		}
	})

	table.SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
		switch event.Key() {
		case tcell.KeyEscape:
			onCancel()
			return nil
		case tcell.KeyRune:
			switch event.Rune() {
			case '1':
				callback(formats[0])
				return nil
			case '2':
				callback(formats[1])
				return nil
			case '3':
				callback(formats[2])
				return nil
			case '4':
				if len(formats) > 3 {
					callback(formats[3])
					return nil
				}
			}
		}
		return event
	})

	frame := tview.NewFrame(table).SetBorders(0, 0, 0, 0, 0, 0)
	frame.SetBorder(true)
	frame.SetBorderColor(theme.ColorDialogBorder)
	frame.SetBackgroundColor(theme.ColorDialogBg)
	frame.SetTitle(" Format ")
	frame.SetTitleColor(theme.ColorHeaderFg)

	rows := len(formats) + 2
	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(nil, 0, 1, false).
		AddItem(tview.NewFlex().SetDirection(tview.FlexColumn).
			AddItem(nil, 0, 1, false).
			AddItem(frame, 20, 0, true).
			AddItem(nil, 0, 1, false),
			rows, 0, true).
		AddItem(nil, 0, 1, false)

	pages.AddPage("format", flex, true, true)
}
