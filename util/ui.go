package util

import (
	"strings"

	"github.com/ayntgl/discordgo"
	"github.com/rivo/tview"
)

// GetNodeByReference walks the root `*TreeNode` of the given `*TreeView` *treeView* and returns the TreeNode whose reference is equal to the given reference *r*. If the `*TreeNode` is not found, `nil` is returned instead.
func GetNodeByReference(treeView *tview.TreeView, r interface{}) (mn *tview.TreeNode) {
	treeView.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}

// ChannelToString constructs a string representation of the given channel. The string representation may vary for different channel types.
func ChannelToString(c *discordgo.Channel) string {
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

// HasKeybinding returns a boolean that indicates whether the given keybinding string representation *k* is in the slice *ks*.
func HasKeybinding(ks []string, k string) bool {
	for _, repr := range ks {
		if repr == k {
			return true
		}
	}

	return false
}
