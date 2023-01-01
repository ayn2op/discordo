package main

import (
	"log"
	"sort"

	"github.com/diamondburned/arikawa/v3/discord"
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

	gt.SetBorder(cfg.Theme.GuildsTree.Border)

	padding := cfg.Theme.GuildsTree.BorderPadding
	gt.SetBorderPadding(padding[0], padding[1], padding[2], padding[3])

	return gt
}

func (gt *GuildsTree) newGuildFromID(n *tview.TreeNode, gid discord.GuildID) error {
	g, err := discordState.Cabinet.Guild(gid)
	if err != nil {
		return err
	}

	gn := tview.NewTreeNode(g.Name)
	gn.SetReference(g.ID)
	n.AddChild(gn)
	return nil
}

func (gt *GuildsTree) newChannel(n *tview.TreeNode, c discord.Channel) {
	cn := tview.NewTreeNode(gt.channelToString(c))
	cn.SetReference(c.ID)
	n.AddChild(cn)
}

func (gt *GuildsTree) channelToString(c discord.Channel) string {
	switch c.Type {
	case discord.GuildText:
		return "#" + c.Name
	case discord.GuildVoice:
		return "v-" + c.Name
	case discord.GuildNews:
		return "n-" + c.Name
	case discord.GuildStore:
		return "s-" + c.Name
	default:
		return c.Name
	}
}

func (gt *GuildsTree) onSelected(n *tview.TreeNode) {
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

		// Orphan (top-level) channels
		for _, c := range cs {
			if c.Type != discord.GuildCategory && !c.ParentID.IsValid() {
				gt.newChannel(n, c)
			}
		}

		// Category channels
	CATEGORY:
		for _, c := range cs {
			if c.Type == discord.GuildCategory {
				for _, nestedChannel := range cs {
					if nestedChannel.ParentID == c.ID {
						gt.newChannel(n, c)
						continue CATEGORY
					}
				}

				gt.newChannel(n, c)
			}
		}

		// Children (category-bound) channels
		for _, c := range cs {
			if c.Type != discord.GuildCategory && c.ParentID.IsValid() {
				var parent *tview.TreeNode
				n.Walk(func(node, _ *tview.TreeNode) bool {
					if node.GetReference() == c.ParentID {
						parent = node
						return false
					}

					return true
				})

				if parent != nil {
					gt.newChannel(parent, c)
				}
			}
		}
	case discord.ChannelID:
		messagesText.Clear()

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
			if messagesText.newMessage(&ms[i]); err != nil {
				log.Println(err)
				continue
			}
		}

		app.SetFocus(messagesText)
	}
}
