package ui

import "github.com/rivo/tview"

type ChannelsTree struct {
	*tview.TreeView
	*Core
}

func NewChannelsTree(c *Core) *ChannelsTree {
	ct := &ChannelsTree{
		TreeView: tview.NewTreeView(),
		Core:     c,
	}

	ct.SetTopLevel(1)
	ct.SetRoot(tview.NewTreeNode(""))

	ct.SetTitle("Channels")
	ct.SetTitleAlign(tview.AlignLeft)
	ct.SetBorder(true)
	ct.SetBorderPadding(0, 0, 1, 1)

	return ct
}
