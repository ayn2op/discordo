package ui

import (
	"log"
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type ChannelsView struct {
	*tview.TreeView
	selectedChannel *discord.Channel
	core            *Core
}

func newChannelsView(c *Core) *ChannelsView {
	v := &ChannelsView{
		TreeView: tview.NewTreeView(),
		core:     c,
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

func (v *ChannelsView) onSelected(node *tview.TreeNode) {
	v.selectedChannel = nil
	v.core.MessagesView.
		Highlight().
		Clear().
		SetTitle("")
	v.core.InputView.SetText("")

	ref := node.GetReference()
	c, err := v.core.State.Cabinet.Channel(ref.(discord.ChannelID))
	if err != nil {
		return
	}

	// If the channel is a category channel, expand the selected node if it is collapsed, otherwise collapse.
	if c.Type == discord.GuildCategory {
		node.SetExpanded(!node.IsExpanded())
		return
	}

	v.selectedChannel = c
	v.core.App.SetFocus(v.core.InputView)

	title := channelToString(*c)
	if c.Topic != "" {
		title += " - " + parseMarkdown(c.Topic)
	}
	v.core.MessagesView.SetTitle(title)

	go func() {
		// The returned slice will be sorted from latest to oldest.
		ms, err := v.core.State.Messages(c.ID, v.core.Config.MessagesLimit)
		if err != nil {
			log.Println(err)
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			_, err = v.core.MessagesView.Write(buildMessage(v.core, ms[i]))
			if err != nil {
				log.Println(err)
				continue
			}
		}

		v.core.MessagesView.ScrollToEnd()
	}()
}

func (v *ChannelsView) createChannelNode(c discord.Channel) *tview.TreeNode {
	channelNode := tview.NewTreeNode(channelToString(c))
	channelNode.SetReference(c.ID)

	return channelNode
}

func (v *ChannelsView) createPrivateChannelNodes(root *tview.TreeNode) {
	cs, err := v.core.State.Cabinet.PrivateChannels()
	if err != nil {
		log.Println(err)
		return
	}

	sort.Slice(cs, func(i, j int) bool {
		return cs[i].LastMessageID > cs[j].LastMessageID
	})

	for _, c := range cs {
		root.AddChild(v.createChannelNode(c))
	}
}

func (v *ChannelsView) createGuildChannelNodes(root *tview.TreeNode, gID discord.GuildID) {
	cs, err := v.core.State.Cabinet.Channels(gID)
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

func (v *ChannelsView) createOrphanChannelNodes(root *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && (!c.ParentID.IsValid()) {
			root.AddChild(v.createChannelNode(c))
		}
	}
}

func (v *ChannelsView) createCategoryChannelNodes(root *tview.TreeNode, cs []discord.Channel) {
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

func (v *ChannelsView) createChildrenChannelNodes(root *tview.TreeNode, cs []discord.Channel) {
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
