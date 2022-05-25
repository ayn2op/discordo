package ui

import "github.com/rivo/tview"

func NewChannelsTreeView() *tview.TreeView {
	t := tview.NewTreeView()
	t.SetBorder(true)
	t.SetBorderPadding(0, 0, 1, 1)
	t.SetTitle(" Channels ")

	return t
}
