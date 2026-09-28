package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/list"
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
