package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewGuildsTreeView(onGuildsTreeViewSelected func(*tview.TreeNode), theme *util.Theme) (treeV *tview.TreeView) {
	treeV = tview.NewTreeView()
	treeN := tview.NewTreeNode("")
	treeV.
		SetTopLevel(1).
		SetRoot(treeN).
		SetCurrentNode(treeN).
		SetSelectedFunc(onGuildsTreeViewSelected).
		SetBackgroundColor(tcell.GetColor(theme.TreeViewBackground)).
		SetTitle("Guilds").
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}
