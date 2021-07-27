package ui

import (
	"github.com/rivo/tview"
)

func NewMainFlex(guildsDropDown *tview.DropDown, channelsList *tview.List, messagesTextView *tview.TextView) (mainFlex *tview.Flex) {
	midFlex := NewMidFlex(channelsList, messagesTextView)
	mainFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(guildsDropDown, 3, 1, false).
		AddItem(midFlex, 0, 1, false)

	return
}

func NewMidFlex(channelsList *tview.List, messagesTextView *tview.TextView) (midFlex *tview.Flex) {
	midFlex = tview.NewFlex().
		AddItem(channelsList, 20, 1, false).
		AddItem(messagesTextView, 0, 3, false)

	return
}
