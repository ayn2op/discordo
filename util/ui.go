package util

import (
	"strings"

	"github.com/ayntgl/discordgo"
	"github.com/rivo/tview"
)

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
