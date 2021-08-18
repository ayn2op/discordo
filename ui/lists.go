package ui

import "github.com/rivo/tview"

func NewGuildsList() (l *tview.List) {
	l = tview.NewList()
	l.
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Guilds").
		SetTitleAlign(tview.AlignLeft)

	return
}
