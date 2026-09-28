package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/picker"
	"github.com/ayn2op/tview/tree"
)

// SelectionKeybind binds kbs to moving a list's selection.
func SelectionKeybind(kbs config.SelectionKeybinds) func(tview.KeyMsg) (list.Action, bool) {
	return func(key tview.KeyMsg) (list.Action, bool) {
		switch {
		case keybind.Matches(key, kbs.SelectUp.Keybind):
			return list.ActionSelectUp, true
		case keybind.Matches(key, kbs.SelectDown.Keybind):
			return list.ActionSelectDown, true
		case keybind.Matches(key, kbs.SelectTop.Keybind):
			return list.ActionSelectTop, true
		case keybind.Matches(key, kbs.SelectBottom.Keybind):
			return list.ActionSelectBottom, true
		}
		return 0, false
	}
}

// ScrollKeybind binds kbs to scrolling a list.
func ScrollKeybind(kbs config.ScrollKeybinds) func(tview.KeyMsg) (list.Action, bool) {
	return func(key tview.KeyMsg) (list.Action, bool) {
		switch {
		case keybind.Matches(key, kbs.ScrollUp.Keybind):
			return list.ActionScrollUp, true
		case keybind.Matches(key, kbs.ScrollDown.Keybind):
			return list.ActionScrollDown, true
		case keybind.Matches(key, kbs.ScrollTop.Keybind):
			return list.ActionScrollTop, true
		case keybind.Matches(key, kbs.ScrollBottom.Keybind):
			return list.ActionScrollBottom, true
		}
		return 0, false
	}
}

// TreeKeybind binds kbs to moving through a tree and selecting its nodes.
func TreeKeybind(kbs config.GuildsTreeKeybinds) func(tview.KeyMsg) (tree.Action, bool) {
	return func(key tview.KeyMsg) (tree.Action, bool) {
		switch {
		case keybind.Matches(key, kbs.SelectUp.Keybind):
			return tree.ActionUp, true
		case keybind.Matches(key, kbs.SelectDown.Keybind):
			return tree.ActionDown, true
		case keybind.Matches(key, kbs.SelectTop.Keybind):
			return tree.ActionTop, true
		case keybind.Matches(key, kbs.SelectBottom.Keybind):
			return tree.ActionBottom, true
		case keybind.Matches(key, kbs.MoveToParentNode.Keybind):
			return tree.ActionMoveToParent, true
		case keybind.Matches(key, kbs.SelectCurrent.Keybind):
			return tree.ActionSelect, true
		}
		return 0, false
	}
}

func pickerKeybind(kbs config.PickerKeybinds) func(tview.KeyMsg) (picker.Action, bool) {
	return func(key tview.KeyMsg) (picker.Action, bool) {
		switch {
		case keybind.Matches(key, kbs.Select.Keybind):
			return picker.ActionSelect, true
		case keybind.Matches(key, kbs.Cancel.Keybind):
			return picker.ActionCancel, true
		}
		return 0, false
	}
}
