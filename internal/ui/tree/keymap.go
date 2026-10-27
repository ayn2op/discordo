package tree

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/tree"
	"github.com/ayn2op/tview/viewport"
)

// SelectHelp returns the up and down keybinds that move the selection away from current.
func SelectHelp(root, current *tree.Node, keybinds config.TreeKeybinds) []keybind.Keybind {
	var help []keybind.Keybind
	if canSelectUp(root, current) {
		help = append(help, keybinds.SelectUp.Keybind)
	}
	if canSelectDown(root, current) {
		help = append(help, keybinds.SelectDown.Keybind)
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

func treeKeybind(keybinds config.TreeKeybinds) func(tview.KeyMsg) tree.Action {
	return func(key tview.KeyMsg) tree.Action {
		switch {
		case keybind.Matches(key, keybinds.SelectUp.Keybind):
			return tree.ActionUp
		case keybind.Matches(key, keybinds.SelectDown.Keybind):
			return tree.ActionDown
		case keybind.Matches(key, keybinds.SelectTop.Keybind):
			return tree.ActionTop
		case keybind.Matches(key, keybinds.SelectBottom.Keybind):
			return tree.ActionBottom
		case keybind.Matches(key, keybinds.MoveToParentNode.Keybind):
			return tree.ActionMoveToParent
		case keybind.Matches(key, keybinds.SelectCurrent.Keybind):
			return tree.ActionSelect
		}
		return tree.ActionNone
	}
}

func scrollKeybind(keybinds config.TreeKeybinds) func(tview.KeyMsg) viewport.Action {
	scroll := ui.ScrollKeybind(keybinds.ScrollKeybinds)
	return func(key tview.KeyMsg) viewport.Action {
		switch {
		case keybind.Matches(key, keybinds.ScrollLeft.Keybind):
			return viewport.ActionLeft
		case keybind.Matches(key, keybinds.ScrollRight.Keybind):
			return viewport.ActionRight
		}
		return scroll(key)
	}
}
