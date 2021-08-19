package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

// NewGuildsTreeView creates and returns a new guilds treeview.
func NewGuildsTreeView(onGuildsTreeViewSelected func(*tview.TreeNode)) (treeV *tview.TreeView) {
	treeN := tview.NewTreeNode("")
	treeV = tview.NewTreeView()
	treeV.
		SetTopLevel(1).
		SetRoot(treeN).
		SetSelectedFunc(onGuildsTreeViewSelected).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Guilds").
		SetTitleAlign(tview.AlignLeft)

	return
}

// NewChannelsTreeView creates and returns a new channels treeview.
func NewChannelsTreeView(onChannelsTreeViewSelected func(*tview.TreeNode)) (treeV *tview.TreeView) {
	treeN := tview.NewTreeNode("")
	treeV = tview.NewTreeView()
	treeV.
		SetTopLevel(1).
		SetRoot(treeN).
		SetSelectedFunc(onChannelsTreeViewSelected).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitle("Channels").
		SetTitleAlign(tview.AlignLeft)

	return
}

// NewTextChannelTreeNode creates and returns a new text channel treenode.
func NewTextChannelTreeNode(c discord.Channel) (n *tview.TreeNode) {
	n = tview.NewTreeNode("[::d]#" + c.Name + "[::-]").
		SetReference(c.ID)

	return
}

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
