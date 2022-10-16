package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type GuildsView struct {
	*tview.TreeView

	app *Application
}

func newGuildsView(app *Application) *GuildsView {
	v := &GuildsView{
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

func (v *GuildsView) onSelected(node *tview.TreeNode) {
	v.app.view.ChannelsView.selected = nil
	v.app.view.MessagesView.selected = -1
	rootNode := v.app.view.ChannelsView.GetRoot()
	rootNode.ClearChildren()
	v.app.view.MessagesView.
		Highlight().
		Clear().
		SetTitle("")
	v.app.view.InputView.SetText("")

	// If the selected node has children (guild folder), expand the selected node if it is collapsed, otherwise collapse.
	if len(node.GetChildren()) != 0 {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	ref := node.GetReference()
	// If the reference of the selected node is nil, it must be the direct messages node.
	if ref == nil {
		v.app.view.ChannelsView.createPrivateChannelNodes(rootNode)
	} else { // Guild
		v.app.view.ChannelsView.createGuildChannelNodes(rootNode, ref.(discord.GuildID))
	}

	v.app.view.ChannelsView.SetCurrentNode(rootNode)
	v.app.SetFocus(v.app.view.ChannelsView)
}
