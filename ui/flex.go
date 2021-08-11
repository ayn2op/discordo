package ui

import (
	"github.com/rivo/tview"
)

func NewMainFlex(treeV *tview.TreeView, textV *tview.TextView, i *tview.InputField) *tview.Flex {
	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(textV, 0, 1, false).
		AddItem(i, 3, 1, false)
	mainFlex := tview.NewFlex().
		AddItem(treeV, 25, 1, false).
		AddItem(rightFlex, 0, 1, false)

	return mainFlex
}
