package ui

import (
	"github.com/rivo/tview"
)

func NewChannelsTreeView(onChannelsTreeViewSelected func(*tview.TreeNode)) (treeV *tview.TreeView) {
	treeV = tview.NewTreeView()
	treeN := tview.NewTreeNode("")
	treeV.
		SetTopLevel(1).
		SetRoot(treeN).
		SetCurrentNode(treeN).
		SetSelectedFunc(onChannelsTreeViewSelected).
		SetTitle("Channels").
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return
}
