package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var guildsDropDownBackgroundColor = tcell.GetColor("#1C1E26")

func NewGuildsDropDown(onGuildsDropDownSelected func(text string, index int)) *tview.DropDown {
	guildsDropDown := tview.NewDropDown().
		SetLabel("Guild: ").
		SetSelectedFunc(onGuildsDropDownSelected)
	guildsDropDown.
		SetFieldBackgroundColor(guildsDropDownBackgroundColor).
		SetBackgroundColor(guildsDropDownBackgroundColor).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return guildsDropDown
}
