package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewGuildsDropDown(onGuildsDropDownSelected func(text string, index int), theme *util.Theme) (guildsDropDown *tview.DropDown) {
	guildsDropDown = tview.NewDropDown().
		SetLabel("Guild: ").
		SetSelectedFunc(onGuildsDropDownSelected)
	guildsDropDown.
		SetLabelColor(tcell.GetColor(theme.DropDownForeground)).
		SetFieldBackgroundColor(tcell.GetColor(theme.DropDownBackground)).
		SetFieldTextColor(tcell.GetColor(theme.DropDownForeground)).
		SetBackgroundColor(tcell.GetColor(theme.DropDownBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}
