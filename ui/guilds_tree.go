package ui

import (
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView
	app *App
}

func NewGuildsTree(app *App) *GuildsTree {
	gt := &GuildsTree{
		TreeView: tview.NewTreeView(),
		app:      app,
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
	gt.app.ChannelsTree.SelectedChannel = nil
	gt.app.MessagesPanel.SelectedMessage = -1
	rootNode := gt.app.ChannelsTree.GetRoot()
	rootNode.ClearChildren()
	gt.app.MessagesPanel.
		Highlight().
		Clear().
		SetTitle("")
	gt.app.MessageInputField.SetText("")

	// If the selected node has children (guild folder), expand the selected node if it is collapsed, otherwise collapse.
	if len(node.GetChildren()) != 0 {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	ref := node.GetReference()
	// If the reference of the selected node is nil, it must be the direct messages node.
	if ref == nil {
		cs, err := gt.app.State.Cabinet.PrivateChannels()
		if err != nil {
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].LastMessageID > cs[j].LastMessageID
		})

		for _, c := range cs {
			rootNode.AddChild(gt.app.ChannelsTree.createChannelNode(c))
		}
	} else { // Guild
		cs, err := gt.app.State.Cabinet.Channels(ref.(discord.GuildID))
		if err != nil {
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		gt.app.ChannelsTree.createOrphanChannelNodes(rootNode, cs)
		gt.app.ChannelsTree.createCategoryChannelNodes(rootNode, cs)
		gt.app.ChannelsTree.createChildrenChannelNodes(rootNode, cs)
	}

	gt.app.ChannelsTree.SetCurrentNode(rootNode)
	gt.app.SetFocus(gt.app.ChannelsTree)
}
