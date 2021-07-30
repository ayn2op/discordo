package ui

import (
	"github.com/rivo/tview"
)

func NewMainFlex(guildsDropDown *tview.DropDown, channelsTreeView *tview.TreeView, messagesTextView *tview.TextView) (mainFlex *tview.Flex) {
	midFlex := tview.NewFlex().
		AddItem(channelsTreeView, 20, 1, false).
		AddItem(messagesTextView, 0, 3, false)
	mainFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(guildsDropDown, 3, 1, false).
		AddItem(midFlex, 0, 1, false)

	return
}
