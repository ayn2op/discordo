package ui

import (
	"sort"

	dsc "github.com/diamondburned/arikawa/v3/discord"
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
	gt.app.SelectedChannel = nil
	gt.app.SelectedMessage = -1
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
			channelNode := tview.NewTreeNode(c.Name)
			channelNode.SetReference(c.ID)
			rootNode.AddChild(channelNode)
		}
	} else { // Guild
		cs, err := gt.app.State.Cabinet.Channels(ref.(dsc.GuildID))
		if err != nil {
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		for _, c := range cs {
			if (c.Type == dsc.GuildText || c.Type == dsc.GuildNews) && (!c.ParentID.IsValid()) {
				channelNode := tview.NewTreeNode(channelToString(c))
				channelNode.SetReference(c.ID)
				rootNode.AddChild(channelNode)
			}
		}

	CATEGORY:
		for _, c := range cs {
			if c.Type == dsc.GuildCategory {
				for _, nestedChannel := range cs {
					if nestedChannel.ParentID == c.ID {
						channelNode := tview.NewTreeNode(c.Name)
						channelNode.SetReference(c.ID)
						rootNode.AddChild(channelNode)
						continue CATEGORY
					}
				}

				channelNode := tview.NewTreeNode(channelToString(c))
				channelNode.SetReference(c.ID)
				rootNode.AddChild(channelNode)
			}
		}

		for _, c := range cs {
			if (c.Type == dsc.GuildText || c.Type == dsc.GuildNews) && (c.ParentID.IsValid()) {
				var parentNode *tview.TreeNode
				rootNode.Walk(func(node, _ *tview.TreeNode) bool {
					if node.GetReference() == c.ParentID {
						parentNode = node
						return false
					}

					return true
				})

				if parentNode != nil {
					channelNode := tview.NewTreeNode(channelToString(c))
					channelNode.SetReference(c.ID)
					parentNode.AddChild(channelNode)
				}
			}
		}
	}

	gt.app.ChannelsTree.SetCurrentNode(rootNode)
	gt.app.SetFocus(gt.app.ChannelsTree)
}
