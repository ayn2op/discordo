package main

import (
	"sort"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/util"
	"github.com/rivo/tview"
)

func createPrivateChannels(n *tview.TreeNode) {
	cs := session.State.PrivateChannels
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].LastMessageID > cs[j].LastMessageID
	})

	for _, c := range cs {
		var tag string
		if util.ChannelIsUnread(session.State, c) {
			tag = "[::b]"
		} else {
			tag = "[::d]"
		}

		cn := tview.NewTreeNode(tag + util.ChannelToString(c) + "[::-]").
			SetReference(c.ID)
		n.AddChild(cn)
	}
}

func createGuilds(n *tview.TreeNode) {
	gs := session.State.Guilds
	sort.Slice(gs, func(a, b int) bool {
		found := false
		for _, gID := range session.State.Settings.GuildPositions {
			if found {
				if gID == gs[b].ID {
					return true
				}
			} else {
				if gID == gs[a].ID {
					found = true
				}
			}
		}

		return false
	})

	for _, g := range gs {
		gn := tview.NewTreeNode(g.Name).Collapse()
		n.AddChild(gn)

		cs := g.Channels
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		// Top-level channels
		createTopLevelChannelsTreeNodes(gn, cs)
		// Category channels
		createCategoryChannelsTreeNodes(gn, cs)
		// Second-level channels
		createSecondLevelChannelsTreeNodes(cs)
	}
}

func createTopLevelChannelsTreeNodes(
	n *tview.TreeNode,
	cs []*discordgo.Channel,
) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID == "") {
			p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID)
			if err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
				continue
			}

			var tag string
			if util.ChannelIsUnread(session.State, c) {
				tag = "[::b]"
			} else {
				tag = "[::d]"
			}

			cn := tview.NewTreeNode(tag + util.ChannelToString(c) + "[::-]").
				SetReference(c.ID)
			n.AddChild(cn)
			continue
		}
	}
}

func createCategoryChannelsTreeNodes(
	n *tview.TreeNode,
	cs []*discordgo.Channel,
) {
CategoryLoop:
	for _, c := range cs {
		if c.Type == discordgo.ChannelTypeGuildCategory {
			p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID)
			if err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
				continue
			}

			for _, child := range cs {
				if child.ParentID == c.ID {
					cn := tview.NewTreeNode(c.Name).
						SetReference(c.ID)
					n.AddChild(cn)
					continue CategoryLoop
				}
			}

			cn := tview.NewTreeNode(c.Name).
				SetReference(c.ID)
			n.AddChild(cn)
		}
	}
}

func createSecondLevelChannelsTreeNodes(cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID != "") {
			p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID)
			if err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
				continue
			}

			var tag string
			if util.ChannelIsUnread(session.State, c) {
				tag = "[::b]"
			} else {
				tag = "[::d]"
			}

			pn := util.GetTreeNodeByReference(channelsTree, c.ParentID)
			if pn != nil {
				cn := tview.NewTreeNode(tag + util.ChannelToString(c) + "[::-]").
					SetReference(c.ID)
				pn.AddChild(cn)
			}
		}
	}
}
