package ui

import (
	"github.com/rivo/tview"
)

func NewGuildsList() *tview.List {
	l := tview.NewList()
	l.ShowSecondaryText(false)
	l.SetBorder(true)
	l.SetBorderPadding(0, 0, 1, 1)
	l.SetTitle(" Guilds ")

	return l
}
