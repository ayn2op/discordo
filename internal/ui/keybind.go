package ui

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/viewport"
)

// SelectionKeybind binds kbs to moving a list's selection.
func SelectionKeybind(kbs config.SelectionKeybinds) func(tview.KeyMsg) list.Action {
	return func(key tview.KeyMsg) list.Action {
		switch {
		case keybind.Matches(key, kbs.SelectUp.Keybind):
			return list.ActionSelectUp
		case keybind.Matches(key, kbs.SelectDown.Keybind):
			return list.ActionSelectDown
		case keybind.Matches(key, kbs.SelectTop.Keybind):
			return list.ActionSelectTop
		case keybind.Matches(key, kbs.SelectBottom.Keybind):
			return list.ActionSelectBottom
		}
		return list.ActionNone
	}
}

// ScrollKeybind binds kbs to scrolling a viewport.
func ScrollKeybind(kbs config.ScrollKeybinds) func(tview.KeyMsg) viewport.Action {
	return func(key tview.KeyMsg) viewport.Action {
		switch {
		case keybind.Matches(key, kbs.ScrollUp.Keybind):
			return viewport.ActionUp
		case keybind.Matches(key, kbs.ScrollDown.Keybind):
			return viewport.ActionDown
		case keybind.Matches(key, kbs.ScrollTop.Keybind):
			return viewport.ActionTop
		case keybind.Matches(key, kbs.ScrollBottom.Keybind):
			return viewport.ActionBottom
		}
		return viewport.ActionNone
	}
}
