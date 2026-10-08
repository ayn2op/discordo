// Package tree holds what discordo's trees share.
package tree

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/tree"
	"github.com/ayn2op/tview/viewport"
	"github.com/gdamore/tcell/v3"
)

// ScrollMsg scrolls the tree, before passing on Msg if the tree sent one.
type ScrollMsg struct {
	Change viewport.Change
	Msg    tview.Msg
}

// New returns a boxed tree of the nodes under root in a viewport.
func New(root *tree.Node, selectionState tree.SelectionState, scrollState viewport.ScrollState, cfg *config.Config, theme config.TreeThemeConfig, graphics bool, kbs config.TreeKeybinds, focused bool, onChange func(tree.Change) tview.Msg, onSelect func(*tree.Node) tview.Msg) box.Widget {
	set := cfg.UI.Border.NormalSet.BorderSet
	if focused {
		set = cfg.UI.Border.ActiveSet.BorderSet
	}
	t := tree.New(root, selectionState).
		TopLevel(1).
		Markers(tree.Markers{
			Expanded:  cfg.Sidebar.Markers.Expanded,
			Collapsed: cfg.Sidebar.Markers.Collapsed,
			Leaf:      cfg.Sidebar.Markers.Leaf,
		}).
		Graphics(graphics).
		GraphicsSet(set).
		GraphicsStyle(tcell.StyleDefault.Foreground(tcell.GetColor(theme.GraphicsColor))).
		Keybind(bindKeys(kbs)).
		Focused(focused).
		OnChange(onChange).
		OnSelect(onSelect)
	v := viewport.New(t, scrollState).
		Target(t.Target).
		ContentWidth(t.RowsWidth).
		OnChildMsg(func(c viewport.Change, msg tview.Msg) tview.Msg { return ScrollMsg{Change: c, Msg: msg} }).
		Axes(layout.Axes{Width: true, Height: true}).
		Keybind(bindScrollKeys(kbs)).
		Focused(focused).
		OnChange(func(c viewport.Change) tview.Msg { return ScrollMsg{Change: c} })
	return ui.Box(v, cfg, focused)
}

func bindKeys(kbs config.TreeKeybinds) func(tview.KeyMsg) tree.Action {
	return func(key tview.KeyMsg) tree.Action {
		switch {
		case keybind.Matches(key, kbs.SelectUp.Keybind):
			return tree.ActionUp
		case keybind.Matches(key, kbs.SelectDown.Keybind):
			return tree.ActionDown
		case keybind.Matches(key, kbs.SelectTop.Keybind):
			return tree.ActionTop
		case keybind.Matches(key, kbs.SelectBottom.Keybind):
			return tree.ActionBottom
		case keybind.Matches(key, kbs.MoveToParentNode.Keybind):
			return tree.ActionMoveToParent
		case keybind.Matches(key, kbs.SelectCurrent.Keybind):
			return tree.ActionSelect
		}
		return tree.ActionNone
	}
}

func bindScrollKeys(kbs config.TreeKeybinds) func(tview.KeyMsg) viewport.Action {
	scroll := ui.ScrollKeybind(kbs.ScrollKeybinds)
	return func(key tview.KeyMsg) viewport.Action {
		switch {
		case keybind.Matches(key, kbs.ScrollLeft.Keybind):
			return viewport.ActionLeft
		case keybind.Matches(key, kbs.ScrollRight.Keybind):
			return viewport.ActionRight
		}
		return scroll(key)
	}
}
