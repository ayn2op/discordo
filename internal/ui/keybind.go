package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/viewport"
)

// SelectionKeybind binds keybinds to moving a list's selection.
func SelectionKeybind(keybinds config.SelectionKeybinds) func(tview.KeyMsg) list.Action {
	return func(key tview.KeyMsg) list.Action {
		switch {
		case keybind.Matches(key, keybinds.SelectUp.Keybind):
			return list.ActionSelectUp
		case keybind.Matches(key, keybinds.SelectDown.Keybind):
			return list.ActionSelectDown
		case keybind.Matches(key, keybinds.SelectTop.Keybind):
			return list.ActionSelectTop
		case keybind.Matches(key, keybinds.SelectBottom.Keybind):
			return list.ActionSelectBottom
		}
		return list.ActionNone
	}
}

// ScrollKeybind binds keybinds to scrolling a viewport.
func ScrollKeybind(keybinds config.ScrollKeybinds) func(tview.KeyMsg) viewport.Action {
	return func(key tview.KeyMsg) viewport.Action {
		switch {
		case keybind.Matches(key, keybinds.ScrollUp.Keybind):
			return viewport.ActionUp
		case keybind.Matches(key, keybinds.ScrollDown.Keybind):
			return viewport.ActionDown
		case keybind.Matches(key, keybinds.ScrollTop.Keybind):
			return viewport.ActionTop
		case keybind.Matches(key, keybinds.ScrollBottom.Keybind):
			return viewport.ActionBottom
		}
		return viewport.ActionNone
	}
}
