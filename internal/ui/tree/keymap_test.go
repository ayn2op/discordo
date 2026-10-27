package tree

import (
	"testing"

	"github.com/ayn2op/tview/tree"
)

// selectTree returns a root with first, then last, which holds child and is collapsed if collapsed is set.
func selectTree(collapsed bool) (root, first, last, child *tree.Node) {
	root = tree.NewNode("")
	first, last, child = tree.NewNode("first"), tree.NewNode("last"), tree.NewNode("child")
	root.AddChild(first).AddChild(last.AddChild(child))
	if collapsed {
		last.Collapse()
	}
	return root, first, last, child
}

func TestCanSelectUp(t *testing.T) {
	root, first, last, _ := selectTree(true)
	expanded, _, expandedLast, expandedChild := selectTree(false)
	for _, test := range []struct {
		name    string
		root    *tree.Node
		current *tree.Node
		want    bool
	}{
		{"none", root, nil, false},
		{"first", root, first, false},
		{"collapsed last", root, last, true},
		{"expanded last", expanded, expandedLast, true},
		{"last child", expanded, expandedChild, true},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := canSelectUp(test.root, test.current); got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}

func TestCanSelectDown(t *testing.T) {
	root, first, last, _ := selectTree(true)
	expanded, _, expandedLast, expandedChild := selectTree(false)
	for _, test := range []struct {
		name    string
		root    *tree.Node
		current *tree.Node
		want    bool
	}{
		{"none", root, nil, true},
		{"first", root, first, true},
		{"collapsed last", root, last, false},
		{"expanded last", expanded, expandedLast, true},
		{"last child", expanded, expandedChild, false},
		{"empty tree", tree.NewNode(""), nil, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := canSelectDown(test.root, test.current); got != test.want {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}
