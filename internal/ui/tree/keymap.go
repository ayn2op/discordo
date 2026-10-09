package tree

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/tree"
)

// SelectHelp returns the up and down keybinds that move the selection away from current.
func SelectHelp(root, current *tree.Node, kbs config.TreeKeybinds) []keybind.Keybind {
	var help []keybind.Keybind
	if canSelectUp(root, current) {
		help = append(help, kbs.SelectUp.Keybind)
	}
	if canSelectDown(root, current) {
		help = append(help, kbs.SelectDown.Keybind)
	}
	return help
}

// canSelectUp reports whether a node is shown above current.
func canSelectUp(root, current *tree.Node) bool {
	children := root.Children()
	return current != nil && len(children) > 0 && current != children[0]
}

// canSelectDown reports whether a node is shown below current.
func canSelectDown(root, current *tree.Node) bool {
	last := root
	for last.Expanded() && len(last.Children()) > 0 {
		last = last.Children()[len(last.Children())-1]
	}
	return last != root && current != last
}
