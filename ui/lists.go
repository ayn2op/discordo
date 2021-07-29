package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewChannelsList(onChannelsListSelected func(i int, mainText string, secondaryText string, _ rune), theme *util.Theme) (channelsList *tview.List) {
	channelsList = tview.NewList().
		ShowSecondaryText(false).
		SetSelectedFunc(onChannelsListSelected)
	channelsList.
		SetMainTextColor(tcell.GetColor(theme.ListMainTextForeground)).
		SetSelectedTextColor(tcell.GetColor(theme.ListSelectedForeground)).
		SetSelectedBackgroundColor(tcell.GetColor(theme.ListBackground)).
		SetBackgroundColor(tcell.GetColor(theme.ListBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}
