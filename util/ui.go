package util

import (
	"github.com/ayntgl/discordgo"
	"github.com/rivo/tview"
)

// GetTreeNodeByReference walks the root `*TreeNode` of the given `*TreeView` *treeView* and returns the TreeNode whose reference is equal to the given reference *r*. If the `*TreeNode` is not found, `nil` is returned instead.
func GetTreeNodeByReference(treeView *tview.TreeView, r interface{}) (mn *tview.TreeNode) {
	treeView.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}

// CreateTopLevelChannelsNodes builds and creates `*tview.TreeNode`s for top-level (channels that have an empty parent ID and of type GUILD_TEXT, GUILD_NEWS) channels. If the client user does not have the VIEW_CHANNEL permission for a channel, the channel is excluded from the parent.
func CreateTopLevelChannelsNodes(treeView *tview.TreeView, s *discordgo.State, n *tview.TreeNode, cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID == "") {
			if !HasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			n.AddChild(CreateChannelNode(s, c))
			continue
		}
	}
}

// CreateCategoryChannelsNodes builds and creates `*tview.TreeNode`s for category (type: GUILD_CATEGORY) channels. If the client user does not have the VIEW_CHANNEL permission for a channel, the channel is excluded from the parent.
func CreateCategoryChannelsNodes(treeView *tview.TreeView, s *discordgo.State, n *tview.TreeNode, cs []*discordgo.Channel) {
CategoryLoop:
	for _, c := range cs {
		if c.Type == discordgo.ChannelTypeGuildCategory {
			if !HasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			for _, child := range cs {
				if child.ParentID == c.ID {
					n.AddChild(CreateChannelNode(s, c))
					continue CategoryLoop
				}
			}

			n.AddChild(CreateChannelNode(s, c))
		}
	}
}

// CreateSecondLevelChannelsNodes builds and creates `*tview.TreeNode`s for second-level (channels that have a non-empty parent ID and of type GUILD_TEXT, GUILD_NEWS) channels. If the client user does not have the VIEW_CHANNEL permission for a channel, the channel is excluded from the parent.
func CreateSecondLevelChannelsNodes(treeView *tview.TreeView, s *discordgo.State, cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID != "") {
			if !HasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			pn := GetTreeNodeByReference(treeView, c.ParentID)
			if pn != nil {
				pn.AddChild(CreateChannelNode(s, c))
			}
		}
	}
}

// CreateChannelNode builds (encorporates unread channels in bold tag, otherwise dim, etc.) and returns a node according to the type of the given channel *c*.
func CreateChannelNode(s *discordgo.State, c *discordgo.Channel) *tview.TreeNode {
	var cn *tview.TreeNode
	switch c.Type {
	case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews:
		tag := "[::d]"
		if ChannelIsUnread(s, c) {
			tag = "[::b]"
		}

		cn = tview.NewTreeNode(tag + ChannelToString(c) + "[::-]").
			SetReference(c.ID)
	case discordgo.ChannelTypeGuildCategory:
		cn = tview.NewTreeNode(c.Name).
			SetReference(c.ID)
	}

	return cn
}

// HasPermission returns a boolean that indicates whether the client user has the given permission *p* in the given channel ID *cID*.
func HasPermission(s *discordgo.State, cID string, p int64) bool {
	perm, err := s.UserChannelPermissions(s.User.ID, cID)
	if err != nil {
		return false
	}

	return perm&p == p
}

func HasKeybinding(sl []string, s string) bool {
	for _, str := range sl {
		if str == s {
			return true
		}
	}

	return false
}
