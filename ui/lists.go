package ui

import (
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewGuildsList(t *util.Theme) (l *tview.List) {
	l = tview.NewList()
	l.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Guilds").
		SetTitleAlign(tview.AlignLeft)

	return
}
