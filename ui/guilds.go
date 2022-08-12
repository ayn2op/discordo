package ui

import "github.com/rivo/tview"

type GuildsList struct {
	*tview.List
}

func NewGuildsList() *GuildsList {
	l := tview.NewList()
	l.ShowSecondaryText(false)

	l.SetTitle("Guilds")
	l.SetTitleAlign(tview.AlignLeft)
	l.SetBorder(true)
	l.SetBorderPadding(0, 0, 1, 1)

	return &GuildsList{
		List: l,
	}
}
