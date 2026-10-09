package tree

import (
	"testing"

	"github.com/ayn2op/tview/tree"
)

func TestCanSelect(t *testing.T) {
	root := tree.NewNode("")
	first, last, child := tree.NewNode("first"), tree.NewNode("last"), tree.NewNode("child")
	root.AddChild(first).AddChild(last.AddChild(child).Collapse())

	check := func(name string, current *tree.Node, up, down bool) {
		t.Helper()
		if got := canSelectUp(root, current); got != up {
			t.Errorf("%s: canSelectUp = %v, want %v", name, got, up)
		}
		if got := canSelectDown(root, current); got != down {
			t.Errorf("%s: canSelectDown = %v, want %v", name, got, down)
		}
	}

	check("none", nil, false, true)
	check("first", first, false, true)
	check("collapsed last", last, true, false)
	last.Expand()
	check("expanded last", last, true, true)
	check("last child", child, true, false)
	if canSelectDown(tree.NewNode(""), nil) {
		t.Error("empty tree: canSelectDown = true, want false")
	}
}
