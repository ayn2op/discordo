package main

import "github.com/rivo/tview"

type ChannelsTree struct {
	*tview.TreeView
}

func newChannelsTree() *ChannelsTree {
	ct := &ChannelsTree{
		TreeView: tview.NewTreeView(),
	}

	ct.SetBorder(true)
	ct.SetBorderPadding(cfg.BorderPadding())

	return ct
}
