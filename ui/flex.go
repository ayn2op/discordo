package ui

import (
	"github.com/rivo/tview"
)

func NewMainFlex(guildsDropDown *tview.DropDown, channelsList *tview.List, messagesTextView *tview.TextView) (mainFlex *tview.Flex) {
	mainFlex = tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(guildsDropDown, 3, 1, false).
		AddItem(
			tview.NewFlex().
				AddItem(channelsList, 20, 1, false).
				AddItem(messagesTextView, 0, 3, false),
			0,
			1,
			false,
		)

	return
}
