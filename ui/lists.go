package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var channelsListBackgroundColor = tcell.GetColor("#1C1E26")

func NewChannelsList(onChannelsListSelected func(i int, mainText string, secondaryText string, _ rune)) *tview.List {
	channelsList := tview.NewList().
		ShowSecondaryText(false).
		SetSelectedFunc(onChannelsListSelected)
	channelsList.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetTitle("Channels").
		SetBackgroundColor(channelsListBackgroundColor)

	return channelsList
}
