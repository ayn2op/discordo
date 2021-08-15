package ui

import (
	"github.com/rivo/tview"
)

func NewMainFlex(treeV *tview.TreeView, textV *tview.TextView, i *tview.InputField) (mf *tview.Flex) {
	rf := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textV, 0, 1, false).
		AddItem(i, 3, 1, false)
	mf = tview.NewFlex().
		AddItem(treeV, 30, 1, false).
		AddItem(rf, 0, 1, false)

	return mf
}
