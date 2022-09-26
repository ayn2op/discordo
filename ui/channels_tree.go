package ui

import (
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type ChannelsTree struct {
	*tview.TreeView
	SelectedChannel *discord.Channel

	core *Core
}

func NewChannelsTree(c *Core) *ChannelsTree {
	ct := &ChannelsTree{
		TreeView: tview.NewTreeView(),
		core:     c,
	}

	ct.SetRoot(tview.NewTreeNode(""))
	ct.SetTopLevel(1)
	ct.SetSelectedFunc(ct.onSelected)

	ct.SetTitle("Channels")
	ct.SetTitleAlign(tview.AlignLeft)
	ct.SetBorder(true)
	ct.SetBorderPadding(0, 0, 1, 1)

	return ct
}

func (ct *ChannelsTree) onSelected(node *tview.TreeNode) {
	ct.SelectedChannel = nil
	ct.core.MessagesPanel.SelectedMessage = -1
	ct.core.MessagesPanel.
		Highlight().
		Clear().
		SetTitle("")
	ct.core.MessageInput.SetText("")

	ref := node.GetReference()
	c, err := ct.core.State.Cabinet.Channel(ref.(discord.ChannelID))
	if err != nil {
		return
	}

	// If the channel is a category channel, expand the selected node if it is collapsed, otherwise collapse.
	if c.Type == discord.GuildCategory {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	ct.SelectedChannel = c
	ct.core.App.SetFocus(ct.core.MessageInput)

	title := channelToString(*c)
	if c.Topic != "" {
		title += " - " + parseMarkdown(c.Topic)
	}
	ct.core.MessagesPanel.SetTitle(title)

	go func() {
		// The returned slice will be sorted from latest to oldest.
		ms, err := ct.core.State.Messages(c.ID, ct.core.Config.MessagesLimit)
		if err != nil {
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			_, err = ct.core.MessagesPanel.Write(buildMessage(ct.core, ms[i]))
			if err != nil {
				return
			}
		}

		ct.core.MessagesPanel.ScrollToEnd()
	}()
}

func (ct *ChannelsTree) createChannelNode(c discord.Channel) *tview.TreeNode {
	channelNode := tview.NewTreeNode(channelToString(c))
	channelNode.SetReference(c.ID)

	return channelNode
}

func (ct *ChannelsTree) createPrivateChannelNodes(rootNode *tview.TreeNode) {
	cs, err := ct.core.State.Cabinet.PrivateChannels()
	if err != nil {
		return
	}

	sort.Slice(cs, func(i, j int) bool {
		return cs[i].LastMessageID > cs[j].LastMessageID
	})

	for _, c := range cs {
		rootNode.AddChild(ct.createChannelNode(c))
	}
}

func (ct *ChannelsTree) createGuildChannelNodes(rootNode *tview.TreeNode, gID discord.GuildID) {
	cs, err := ct.core.State.Cabinet.Channels(gID)
	if err != nil {
		return
	}

	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Position < cs[j].Position
	})

	ct.createOrphanChannelNodes(rootNode, cs)
	ct.createCategoryChannelNodes(rootNode, cs)
	ct.createChildrenChannelNodes(rootNode, cs)
}

func (ct *ChannelsTree) createOrphanChannelNodes(rootNode *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (!c.ParentID.IsValid()) {
			rootNode.AddChild(ct.createChannelNode(c))
		}
	}
}

func (ct *ChannelsTree) createCategoryChannelNodes(rootNode *tview.TreeNode, cs []discord.Channel) {
CATEGORY:
	for _, c := range cs {
		if c.Type == discord.GuildCategory {
			for _, nestedChannel := range cs {
				if nestedChannel.ParentID == c.ID {
					rootNode.AddChild(ct.createChannelNode(c))
					continue CATEGORY
				}
			}

			rootNode.AddChild(ct.createChannelNode(c))
		}
	}
}

func (ct *ChannelsTree) createChildrenChannelNodes(rootNode *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (c.ParentID.IsValid()) {
			var parentNode *tview.TreeNode
			rootNode.Walk(func(node, _ *tview.TreeNode) bool {
				if node.GetReference() == c.ParentID {
					parentNode = node
					return false
				}

				return true
			})

			if parentNode != nil {
				parentNode.AddChild(ct.createChannelNode(c))
			}
		}
	}
}
