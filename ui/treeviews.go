package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewChannelsTreeView(channelsTreeNode *tview.TreeNode, onChannelsTreeViewSelected func(node *tview.TreeNode), theme *util.Theme) (channelsTreeView *tview.TreeView) {
	channelsTreeView = tview.NewTreeView().
		SetTopLevel(1).
		SetRoot(channelsTreeNode).
		SetCurrentNode(channelsTreeNode).
		SetSelectedFunc(onChannelsTreeViewSelected)
	channelsTreeView.
		SetBackgroundColor(tcell.GetColor(theme.TreeViewBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}

func NewChannelsTreeNode() (channelsTreeNode *tview.TreeNode) {
	channelsTreeNode = tview.NewTreeNode("")
	return
}
