package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewChannelsList(onChannelsListSelected func(i int, mainText string, secondaryText string, _ rune)) (channelsList *tview.List) {
	channelsList = tview.NewList().
		ShowSecondaryText(false).
		SetMainTextColor(tcell.ColorDarkGray).
		SetSelectedTextColor(tcell.ColorWhite).
		SetSelectedBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetSelectedFunc(onChannelsListSelected)
	channelsList.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle("Channels")

	return
}
