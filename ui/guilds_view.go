package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type GuildsView struct {
	*tview.TreeView
	core *Core
}

func newGuildsView(c *Core) *GuildsView {
	v := &GuildsView{
		TreeView: tview.NewTreeView(),
		core:     c,
	}

	root := tview.NewTreeNode("")
	root.AddChild(tview.NewTreeNode("Direct Messages"))

	v.SetRoot(root)
	v.SetTopLevel(1)
	v.SetSelectedFunc(v.onSelected)

	v.SetTitle("Guilds")
	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.SetBorderPadding(0, 0, 1, 1)

	return v
}

func (v *GuildsView) onSelected(node *tview.TreeNode) {
	v.core.ChannelsView.selectedChannel = nil
	v.core.MessagesView.selectedMessage = -1
	rootNode := v.core.ChannelsView.GetRoot()
	rootNode.ClearChildren()
	v.core.MessagesView.
		Highlight().
		Clear().
		SetTitle("")
	v.core.InputView.SetText("")

	// If the selected node has children (guild folder), expand the selected node if it is collapsed, otherwise collapse.
	if len(node.GetChildren()) != 0 {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	ref := node.GetReference()
	// If the reference of the selected node is nil, it must be the direct messages node.
	if ref == nil {
		v.core.ChannelsView.createPrivateChannelNodes(rootNode)
	} else { // Guild
		v.core.ChannelsView.createGuildChannelNodes(rootNode, ref.(discord.GuildID))
	}

	v.core.ChannelsView.SetCurrentNode(rootNode)
	v.core.App.SetFocus(v.core.ChannelsView)
}
