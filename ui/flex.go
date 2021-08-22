package ui

import (
	"github.com/rivo/tview"
)

// NewMainFlex creates and returns a new main flex.
func NewMainFlex(gTreeV *tview.TreeView, cTreeV *tview.TreeView, textV *tview.TextView, i *tview.InputField) *tview.Flex {
	lf := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(gTreeV, 0, 1, false).
		AddItem(cTreeV, 0, 2, false)
	rf := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textV, 0, 1, false).
		AddItem(i, 3, 1, false)
	mf := tview.NewFlex().
		AddItem(lf, 0, 1, false).
		AddItem(rf, 0, 4, false)

	return mf
}
