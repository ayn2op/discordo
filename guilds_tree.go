package main

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView
}

func newGuildsTree() *GuildsTree {
	gt := &GuildsTree{
		TreeView: tview.NewTreeView(),
	}

	root := tview.NewTreeNode("")
	gt.SetRoot(root)
	gt.SetTopLevel(1)
	gt.SetSelectedFunc(gt.onSelected)

	gt.SetBorder(true)
	gt.SetBorderPadding(cfg.BorderPadding())

	return gt
}

func (gt *GuildsTree) newGuild(n *tview.TreeNode, gid discord.GuildID) error {
	g, err := discordState.Cabinet.Guild(gid)
	if err != nil {
		return err
	}

	gn := tview.NewTreeNode(g.Name)
	gn.SetReference(g.ID)
	n.AddChild(gn)
	return nil
}

func (gt *GuildsTree) onSelected(n *tview.TreeNode) {
	ref := n.GetReference()
	if ref == nil {

	}
}
