package ui

import (
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

// NewGuildsList creates and returns a new guilds list.
func NewGuildsList(onGuildsListSelected func(int, string, string, rune), t *util.Theme) (l *tview.List) {
	l = tview.NewList()
	l.
		SetSelectedFunc(onGuildsListSelected).
		ShowSecondaryText(false).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Guilds").
		SetTitleAlign(tview.AlignLeft)

	return
}
