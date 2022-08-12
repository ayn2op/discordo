package ui

import "github.com/rivo/tview"

type ChannelsTree struct {
	*tview.TreeView
}

func NewChannelsTree() *ChannelsTree {
	tv := tview.NewTreeView()

	tv.SetRoot(tview.NewTreeNode(""))

	tv.SetTitle("Channels")
	tv.SetTitleAlign(tview.AlignLeft)
	tv.SetBorder(true)
	tv.SetBorderPadding(0, 0, 1, 1)

	return &ChannelsTree{
		TreeView: tv,
	}
}
