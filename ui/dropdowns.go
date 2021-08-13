package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewGuildsDropDown(onGuildsDropDownSelected func(string, int), theme *util.Theme) *tview.DropDown {
	guildsDropDown := tview.NewDropDown()

	guildsDropDown.
		SetLabel("Guild: ").
		SetSelectedFunc(onGuildsDropDownSelected).
		SetFieldBackgroundColor(tcell.GetColor(theme.DropDownBackground)).
		SetBackgroundColor(tcell.GetColor(theme.DropDownBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return guildsDropDown
}
