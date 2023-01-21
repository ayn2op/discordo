package main

import (
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type GuildsTree struct {
	*tview.TreeView

	root            *tview.TreeNode
	selectedChannel *discord.Channel
}

func newGuildsTree() *GuildsTree {
	gt := &GuildsTree{
		TreeView: tview.NewTreeView(),

		root: tview.NewTreeNode(""),
	}

	gt.SetGraphics(cfg.Theme.GuildsTree.Graphics)
	gt.SetRoot(gt.root)
	gt.SetTopLevel(1)
	gt.SetSelectedFunc(gt.onSelected)

	gt.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))

	gt.SetTitle("Guilds")
	gt.SetTitleColor(tcell.GetColor(cfg.Theme.TitleColor))
	gt.SetTitleAlign(tview.AlignLeft)

	p := cfg.Theme.BorderPadding
	gt.SetBorder(cfg.Theme.Border)
	gt.SetBorderColor(tcell.GetColor(cfg.Theme.BorderColor))
	gt.SetBorderPadding(p[0], p[1], p[2], p[3])

	return gt
}

func (gt *GuildsTree) createGuildFolderNode(parent *tview.TreeNode, gf gateway.GuildFolder) {
	var name string
	if gf.Name != "" {
		name = fmt.Sprintf("[%s]%s[-]", gf.Color.String(), gf.Name)
	} else {
		name = "Folder"
	}

	n := tview.NewTreeNode(name)
	parent.AddChild(n)

	for _, gid := range gf.GuildIDs {
		g, err := discordState.Cabinet.Guild(gid)
		if err != nil {
			log.Println(err)
			continue
		}

		gt.createGuildNode(n, *g)
	}
}

func (gt *GuildsTree) createGuildNode(n *tview.TreeNode, g discord.Guild) {
	gn := tview.NewTreeNode(g.Name)
	gn.SetReference(g.ID)
	n.AddChild(gn)
}

func (gt *GuildsTree) createChannelNode(n *tview.TreeNode, c discord.Channel) {
	cn := tview.NewTreeNode(gt.channelToString(c))
	cn.SetReference(c.ID)
	n.AddChild(cn)
}

func (gt *GuildsTree) channelToString(c discord.Channel) string {
	var s string
	switch c.Type {
	case discord.GuildText:
		s = "#" + c.Name
	case discord.DirectMessage:
		r := c.DMRecipients[0]
		s = r.Tag()
	case discord.GuildVoice:
		s = "v-" + c.Name
	case discord.GroupDM:
		s = c.Name
		// If the name of the channel is empty, use the recipients' tags
		if s == "" {
			rs := make([]string, len(c.DMRecipients))
			for _, r := range c.DMRecipients {
				rs = append(rs, r.Tag())
			}

			s = strings.Join(rs, ", ")
		}
	case discord.GuildNews:
		s = "n-" + c.Name
	case discord.GuildStore:
		s = "s-" + c.Name
	default:
		s = c.Name
	}

	return s
}

func (gt *GuildsTree) createChannelNodes(n *tview.TreeNode, cs []discord.Channel) {
	for _, c := range cs {
		// If the type of the channel is guild category or if the channel is an orphan channel.
		if c.Type == discord.GuildCategory || !c.ParentID.IsValid() {
			gt.createChannelNode(n, c)
		} else {
			var parent *tview.TreeNode
			n.Walk(func(node, _ *tview.TreeNode) bool {
				if node.GetReference() == c.ParentID {
					parent = node
					return false
				}

				return true
			})

			if parent != nil {
				gt.createChannelNode(parent, c)
			}
		}
	}
}

func (gt *GuildsTree) onSelected(n *tview.TreeNode) {
	gt.selectedChannel = nil

	messagesText.reset()
	messageInput.reset()

	if len(n.GetChildren()) != 0 {
		n.SetExpanded(!n.IsExpanded())
		return
	}

	switch ref := n.GetReference().(type) {
	case discord.GuildID:
		cs, err := discordState.Cabinet.Channels(ref)
		if err != nil {
			log.Println(err)
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		gt.createChannelNodes(n, cs)
	case discord.ChannelID:
		c, err := discordState.Cabinet.Channel(ref)
		if err != nil {
			log.Println(err)
			return
		}

		gt.selectedChannel = c
		messagesText.SetTitle(gt.channelToString(*c))

		ms, err := discordState.Messages(ref, cfg.MessagesLimit)
		if err != nil {
			log.Println(err)
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			messagesText.createMessage(&ms[i])
		}

		app.SetFocus(messageInput)
	case nil: // Direct messages
		cs, err := discordState.Cabinet.PrivateChannels()
		if err != nil {
			log.Println(err)
			return
		}

		sort.Slice(cs, func(i, j int) bool {
			return cs[i].LastMessageID < cs[j].LastMessageID
		})

		for _, c := range cs {
			gt.createChannelNode(n, c)
		}
	}
}
