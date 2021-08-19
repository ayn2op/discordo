package util

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

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

func NewTextChannelTreeNode(c discord.Channel) (n *tview.TreeNode) {
	n = tview.NewTreeNode("[::d]#" + c.Name + "[::-]").
		SetReference(c.ID)

	return
}
