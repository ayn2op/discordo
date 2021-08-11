package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewGuildsTreeView(guildsTreeNode *tview.TreeNode, onGuildsTreeViewSelected func(node *tview.TreeNode), theme *util.Theme) *tview.TreeView {
	guildsTreeView := tview.NewTreeView()

	guildsTreeView.
		SetTopLevel(1).
		SetRoot(guildsTreeNode).
		SetCurrentNode(guildsTreeNode).
		SetSelectedFunc(onGuildsTreeViewSelected).
		SetBackgroundColor(tcell.GetColor(theme.TreeViewBackground)).
		SetTitle("Guilds").
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return guildsTreeView
}
