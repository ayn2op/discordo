package main

import (
	"github.com/rivo/tview"
)

type MessagesText struct {
	*tview.TextView
}

func newMessagesText() *MessagesText {
	mt := &MessagesText{
		TextView: tview.NewTextView(),
	}

	mt.SetDynamicColors(true)
	mt.SetRegions(true)

	mt.SetBorder(true)
	mt.SetBorderPadding(cfg.BorderPadding())

	return mt
}
