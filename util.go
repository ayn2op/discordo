package main

import (
	"sort"
	"strings"

	"github.com/ayntgl/discordgo"
	"github.com/rivo/tview"
)

func generateChannelRepr(c *discordgo.Channel) string {
	var repr string
	if c.Name != "" {
		repr = "#" + c.Name
	} else if len(c.Recipients) == 1 {
		rp := c.Recipients[0]
		repr = rp.Username + "#" + rp.Discriminator
	} else {
		rps := make([]string, len(c.Recipients))
		for i, r := range c.Recipients {
			rps[i] = r.Username + "#" + r.Discriminator
		}

		repr = strings.Join(rps, ", ")
	}

	return repr
}

func findByMessageID(mID string) *discordgo.Message {
	for _, m := range selectedChannel.Messages {
		if m.ID == mID {
			return m
		}
	}

	return nil
}

func createPrivateChannels(n *tview.TreeNode) {
	cs := session.State.PrivateChannels
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].LastMessageID > cs[j].LastMessageID
	})

	for _, c := range cs {
		var tag string
		if isUnread(c) {
			tag = "[::b]"
		} else {
			tag = "[::d]"
		}

		cn := tview.NewTreeNode(tag + generateChannelRepr(c) + "[::-]").
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
			if isUnread(c) {
				tag = "[::b]"
			} else {
				tag = "[::d]"
			}

			cn := tview.NewTreeNode(tag + generateChannelRepr(c) + "[::-]").
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
			if isUnread(c) {
				tag = "[::b]"
			} else {
				tag = "[::d]"
			}

			pn := getTreeNodeByReference(c.ParentID)
			if pn != nil {
				cn := tview.NewTreeNode(tag + generateChannelRepr(c) + "[::-]").
					SetReference(c.ID)
				pn.AddChild(cn)
			}
		}
	}
}

func getTreeNodeByReference(r interface{}) (mn *tview.TreeNode) {
	channelsTree.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}

func isUnread(c *discordgo.Channel) bool {
	if c.LastMessageID == "" {
		return false
	}

	for _, rs := range session.State.ReadState {
		if c.ID == rs.ID {
			return c.LastMessageID != rs.LastMessageID
		}
	}

	return false
}

func openInDefaultBrowser(url string) error {
	// source: https://stackoverflow.com/questions/10377243/
	var err error
	switch runtime.GOOS {
		case "linux":
			err = exec.Command("xdg-open", url).Start()
		case "windows", "darwin":
			err = exec.Command("open", url).Start()
		default:
			err = fmt.Errorf("unsupported platform")
	}

	return err
}
