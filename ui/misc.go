package ui

import "github.com/rivo/tview"

// NewMainFlex creates and returns a new main flex.
func NewMainFlex(
	treeV *tview.TreeView,
	textV *tview.TextView,
	i *tview.InputField,
) *tview.Flex {
	rf := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textV, 0, 1, false).
		AddItem(i, 3, 1, false)
	mf := tview.NewFlex().
		AddItem(treeV, 0, 1, false).
		AddItem(rf, 0, 4, false)

	return mf
}
