package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type ChannelsTree struct {
	*tview.TreeView
	app  *App
	root *tview.TreeNode
}

func NewChannelsTreeView(a *App) *ChannelsTree {
	t := &ChannelsTree{
		TreeView: tview.NewTreeView(),
		app:      a,
		root:     tview.NewTreeNode(""),
	}

	// Set the top level to one, that is, do not display the root node, it is a "container" node for all of the channel nodes.
	t.SetTopLevel(1)
	t.SetRoot(t.root)
	t.SetBorder(true)
	t.SetBorderPadding(0, 0, 1, 1)
	t.SetTitle(" Channels ")

	t.SetGraphics(a.config.Theme.ChannelsTree.Graphics)
	t.SetGraphicsColor(tcell.GetColor(a.config.Theme.ChannelsTree.GraphicsForeground))

	return t
}
