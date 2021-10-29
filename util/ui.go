package util

import "github.com/rivo/tview"

// GetTreeNodeByReference walks the root `*TreeNode` of the given `*TreeView` *treeView* and returns the TreeNode whose reference is equal to the given reference *r*. If the `*TreeNode` is not found, `nil` is returned instead.
func GetTreeNodeByReference(treeView *tview.TreeView, r interface{}) (mn *tview.TreeNode) {
	treeView.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}
