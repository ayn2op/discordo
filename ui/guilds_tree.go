package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView

	app *Application
}

func newGuildsTree(app *Application) *GuildsTree {
	v := &GuildsTree{
		TreeView: tview.NewTreeView(),

		app: app,
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

func (v *GuildsTree) onSelected(node *tview.TreeNode) {
	v.app.view.ChannelsTree.selected = nil
	v.app.view.MessagesText.selected = -1
	rootNode := v.app.view.ChannelsTree.GetRoot()
	rootNode.ClearChildren()
	v.app.view.MessagesText.
		Highlight().
		Clear().
		SetTitle("")
	v.app.view.MessageInput.SetText("")

	// If the selected node has children (guild folder), expand the selected node if it is collapsed, otherwise collapse.
	if len(node.GetChildren()) != 0 {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	ref := node.GetReference()
	// If the reference of the selected node is nil, it must be the direct messages node.
	if ref == nil {
		v.app.view.ChannelsTree.createPrivateChannelNodes(rootNode)
	} else { // Guild
		v.app.view.ChannelsTree.createGuildChannelNodes(rootNode, ref.(discord.GuildID))
	}

	v.app.view.ChannelsTree.SetCurrentNode(rootNode)
	v.app.SetFocus(v.app.view.ChannelsTree)
}
