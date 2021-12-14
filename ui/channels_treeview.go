package ui

import "github.com/rivo/tview"

func NewChannelsTreeView() *tview.TreeView {
	v := tview.NewTreeView()
	v.
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return v
}
