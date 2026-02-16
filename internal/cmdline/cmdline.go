package cmdline

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"

	"github.com/feherkaroly/vc/internal/theme"
)

// CmdLine is the command line input at the bottom of the screen.
type CmdLine struct {
	*tview.InputField
	path        string
	executeFunc func(string)
	focusFunc   func(bool)
}

func New() *CmdLine {
	input := tview.NewInputField()
	input.SetFieldBackgroundColor(theme.ColorCmdLineBg)
	input.SetFieldTextColor(theme.ColorCmdLineFg)
	input.SetLabelColor(theme.ColorCmdLineFg)
	input.SetBackgroundColor(theme.ColorCmdLineBg)

	c := &CmdLine{
		InputField: input,
	}

	input.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			text := input.GetText()
			input.SetText("")
			if c.focusFunc != nil {
				c.focusFunc(false)
			}
			if c.executeFunc != nil && text != "" {
				c.executeFunc(text)
			}
		case tcell.KeyEscape:
			input.SetText("")
			if c.focusFunc != nil {
				c.focusFunc(false)
			}
		}
	})

	return c
}

func (c *CmdLine) SetPath(path string) {
	c.path = path
	c.SetLabel(shortenPath(path) + "> ")
}

func (c *CmdLine) SetExecuteFunc(f func(string)) {
	c.executeFunc = f
}

func (c *CmdLine) SetFocusFunc(f func(bool)) {
	c.focusFunc = f
}

func shortenPath(path string) string {
	if len(path) > 40 {
		return "..." + path[len(path)-37:]
	}
	return path
}
