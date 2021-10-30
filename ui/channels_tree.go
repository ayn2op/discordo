package ui

import "github.com/rivo/tview"

func NewChannelsTree() *tview.TreeView {
	treeView := tview.NewTreeView()
	treeView.
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetTitle("Channels").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return treeView
}
