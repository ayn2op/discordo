package ui

import (
	"github.com/rivo/tview"
)

func NewGuildsDropDown(onGuildsDropDownSelected func(text string, index int)) (guildsDropDown *tview.DropDown) {
	guildsDropDown = tview.NewDropDown().
		SetLabel("Guild: ").
		SetSelectedFunc(onGuildsDropDownSelected)
	guildsDropDown.
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}
