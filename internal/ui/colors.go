package ui

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
)

// SetColors sets the terminal's default colors to base's.
func SetColors(base tview.Style) tview.Cmd {
	return cmd.Batch{cmd.SetForegroundColor(base.GetForeground()), cmd.SetBackgroundColor(base.GetBackground())}
}
