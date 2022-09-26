package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView
	core *Core
}

func NewGuildsTree(c *Core) *GuildsTree {
	gt := &GuildsTree{
		TreeView: tview.NewTreeView(),
		core:     c,
	}

	rootNode := tview.NewTreeNode("")
	rootNode.AddChild(tview.NewTreeNode("Direct Messages"))

	gt.SetRoot(rootNode)
	gt.SetTopLevel(1)
	gt.SetSelectedFunc(gt.onSelected)

	gt.SetTitle("Guilds")
	gt.SetTitleAlign(tview.AlignLeft)
	gt.SetBorder(true)
	gt.SetBorderPadding(0, 0, 1, 1)

	return gt
}

func (gt *GuildsTree) onSelected(node *tview.TreeNode) {
	gt.core.ChannelsTree.SelectedChannel = nil
	gt.core.MessagesPanel.SelectedMessage = -1
	rootNode := gt.core.ChannelsTree.GetRoot()
	rootNode.ClearChildren()
	gt.core.MessagesPanel.
		Highlight().
		Clear().
		SetTitle("")
	gt.core.MessageInput.SetText("")

	// If the selected node has children (guild folder), expand the selected node if it is collapsed, otherwise collapse.
	if len(node.GetChildren()) != 0 {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	ref := node.GetReference()
	// If the reference of the selected node is nil, it must be the direct messages node.
	if ref == nil {
		gt.core.ChannelsTree.createPrivateChannelNodes(rootNode)
	} else { // Guild
		gt.core.ChannelsTree.createGuildChannelNodes(rootNode, ref.(discord.GuildID))
	}

	gt.core.ChannelsTree.SetCurrentNode(rootNode)
	gt.core.App.SetFocus(gt.core.ChannelsTree)
}
