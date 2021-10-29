package util

import "github.com/rivo/tview"

func GetTreeNodeByReference(tv *tview.TreeView, r interface{}) (mn *tview.TreeNode) {
	tv.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}
