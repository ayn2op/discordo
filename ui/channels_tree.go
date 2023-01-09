package ui

import (
	"log"
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type ChannelsTree struct {
	*tview.TreeView

	app      *Application
	selected *discord.Channel
}

func newChannelsTree(app *Application) *ChannelsTree {
	v := &ChannelsTree{
		TreeView: tview.NewTreeView(),

		app: app,
	}

	v.SetRoot(tview.NewTreeNode(""))
	v.SetTopLevel(1)
	v.SetSelectedFunc(v.onSelected)

	v.SetTitle("Channels")
	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.SetBorderPadding(0, 0, 1, 1)

	return v
}

func (v *ChannelsTree) onSelected(node *tview.TreeNode) {
	v.selected = nil
	v.app.view.MessagesText.selected = -1
	v.app.view.MessagesText.
		Highlight().
		Clear().
		SetTitle("")
	v.app.view.MessageInput.SetText("")

	ref := node.GetReference()
	if ref == nil {
		log.Println("selected channel reference is nil")
		return
	}

	c, err := v.app.state.Cabinet.Channel(ref.(discord.ChannelID))
	if err != nil {
		log.Println("selected channel not found")
		return
	}

	switch c.Type {
	// If the channel is a category channel, expand the selected node if it is collapsed, otherwise collapse.
	case discord.GuildCategory:
		node.SetExpanded(!node.IsExpanded())
		return

	default:
		v.selected = c

		v.app.view.MessagesText.setTitle(c)
		v.app.SetFocus(v.app.view.MessageInput)

		go v.app.view.MessagesText.loadMessages(c)
	}
}

func (v *ChannelsTree) createChannelNode(c discord.Channel) *tview.TreeNode {
	channelNode := tview.NewTreeNode(channelToString(c))
	channelNode.SetReference(c.ID)

	return channelNode
}

func (v *ChannelsTree) createPrivateChannelNodes(root *tview.TreeNode) {
	cs, err := v.app.state.Cabinet.PrivateChannels()
	if err != nil {
		log.Println(err)
		return
	}

	sort.Slice(cs, func(i, j int) bool {
		idUsed := discord.MessageID(cs[i].ID)
		idUsed2 := discord.MessageID(cs[j].ID)
		if cs[i].LastMessageID.IsValid() {
			idUsed = cs[i].LastMessageID
		}
		if cs[j].LastMessageID.IsValid() {
			idUsed2 = cs[j].LastMessageID
		}
		return idUsed > idUsed2
	})

	for _, c := range cs {
		root.AddChild(v.createChannelNode(c))
	}
}

func (v *ChannelsTree) createGuildChannelNodes(root *tview.TreeNode, gID discord.GuildID) {
	cs, err := v.app.state.Cabinet.Channels(gID)
	if err != nil {
		log.Println(err)
		return
	}

	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Position < cs[j].Position
	})

	v.createOrphanChannelNodes(root, cs)
	v.createCategoryChannelNodes(root, cs)
	v.createChildrenChannelNodes(root, cs)
}

func (v *ChannelsTree) createOrphanChannelNodes(root *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (!c.ParentID.IsValid()) {
			root.AddChild(v.createChannelNode(c))
		}
	}
}

func (v *ChannelsTree) createCategoryChannelNodes(root *tview.TreeNode, cs []discord.Channel) {
CATEGORY:
	for _, c := range cs {
		if c.Type == discord.GuildCategory {
			for _, nestedChannel := range cs {
				if nestedChannel.ParentID == c.ID {
					root.AddChild(v.createChannelNode(c))
					continue CATEGORY
				}
			}

			root.AddChild(v.createChannelNode(c))
		}
	}
}

func (v *ChannelsTree) createChildrenChannelNodes(root *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (c.ParentID.IsValid()) {
			var parentNode *tview.TreeNode
			root.Walk(func(node, _ *tview.TreeNode) bool {
				if node.GetReference() == c.ParentID {
					parentNode = node
					return false
				}

				return true
			})

			if parentNode != nil {
				parentNode.AddChild(v.createChannelNode(c))
			}
		}
	}
}
