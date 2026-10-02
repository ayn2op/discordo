// Package tree holds what discordo's trees share.
package tree

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/tree"
	"github.com/gdamore/tcell/v3"
)

// New returns a boxed tree of the nodes under root, drawn with theme and moved with kbs, that turns changes and selected nodes into the messages onChange and onSelect return.
func New(root *tree.Node, selectionState tree.SelectionState, cfg *config.Config, theme config.CommonTreeTheme, kbs config.TreeKeybinds, focused bool, onChange func(tree.Change) tview.Msg, onSelect func(*tree.Node) tview.Msg) box.Widget {
	set := cfg.Theme.Border.NormalSet.BorderSet
	if focused {
		set = cfg.Theme.Border.ActiveSet.BorderSet
	}
	t := tree.New(root, selectionState).
		TopLevel(1).
		Markers(tree.Markers{
			Expanded:  cfg.Sidebar.Markers.Expanded,
			Collapsed: cfg.Sidebar.Markers.Collapsed,
			Leaf:      cfg.Sidebar.Markers.Leaf,
		}).
		Graphics(theme.Graphics).
		GraphicsSet(set).
		GraphicsStyle(tcell.StyleDefault.Foreground(tcell.GetColor(theme.GraphicsColor))).
		Keybind(bindKeys(kbs)).
		Focused(focused).
		OnChange(onChange).
		OnSelect(onSelect)
	return ui.Box(t, &cfg.Theme, focused)
}

func bindKeys(kbs config.TreeKeybinds) func(tview.KeyMsg) (tree.Action, bool) {
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
