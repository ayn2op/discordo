package ui

import (
	"github.com/rigormorrtiss/discordgo"
	"github.com/rivo/tview"
)

// NewGuildsTreeView creates and returns a new guilds treeview.
func NewGuildsTreeView(onGuildsTreeViewSelected func(*tview.TreeNode)) *tview.TreeView {
	v := tview.NewTreeView()
	v.
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetSelectedFunc(onGuildsTreeViewSelected).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Guilds").
		SetTitleAlign(tview.AlignLeft)

	return v
}

// NewChannelsTreeView creates and returns a new channels treeview.
func NewChannelsTreeView(onChannelsTreeViewSelected func(*tview.TreeNode)) *tview.TreeView {
	v := tview.NewTreeView()
	v.
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetSelectedFunc(onChannelsTreeViewSelected).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Channels").
		SetTitleAlign(tview.AlignLeft)

	return v
}

// NewTextChannelTreeNode creates and returns a new text channel treenode.
func NewTextChannelTreeNode(c *discordgo.Channel) *tview.TreeNode {
	n := tview.NewTreeNode("[::d]#" + c.Name + "[::-]").
		SetReference(c.ID)

	return n
}

// GetTreeNodeByReference gets the TreeNode that has reference r from the given treeview.
func GetTreeNodeByReference(r interface{}, treeV *tview.TreeView) (mn *tview.TreeNode) {
	treeV.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}

// CreateTopLevelTreeNodes creates treenodes for the top-level (orphan) channels.
func CreateTopLevelTreeNodes(rootN *tview.TreeNode, cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) && (c.ParentID == "") {
			cn := NewTextChannelTreeNode(c)
			rootN.AddChild(cn)
			continue
		}
	}
}

// CreateSecondLevelTreeNodes creates treenodes for the second-level (category children) channels.
func CreateSecondLevelTreeNodes(channelsTreeView *tview.TreeView, rootN *tview.TreeNode, cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) && (c.ParentID != "") {
			if pn := GetTreeNodeByReference(c.ParentID, channelsTreeView); pn != nil {
				cn := NewTextChannelTreeNode(c)
				pn.AddChild(cn)
			}
		}
	}
}
