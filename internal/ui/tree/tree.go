// Package tree holds what discordo's trees share.
package tree

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/tree"
	"github.com/ayn2op/tview/viewport"
	"github.com/gdamore/tcell/v3"
)

// ScrollMsg scrolls the tree to ScrollState, before passing on Msg if the tree sent one.
type ScrollMsg struct {
	ScrollState viewport.ScrollState
	Msg         tview.Msg
}

// New returns a boxed tree of the nodes under root in a viewport.
func New(
	root, current *tree.Node,
	scrollState viewport.ScrollState,
	cfg *config.Config,
	theme config.TreeThemeConfig,
	graphics bool,
	kbs config.TreeKeybinds,
	focused bool,
	onChange, onSelect func(*tree.Node) tview.Msg,
) box.Widget {
	set := cfg.UI.Border.NormalSet.BorderSet
	if focused {
		set = cfg.UI.Border.ActiveSet.BorderSet
	}
	t := tree.New(root, current).
		TopLevel(1).
		Markers(tree.Markers{
			Expanded:  cfg.Sidebar.Markers.Expanded,
			Collapsed: cfg.Sidebar.Markers.Collapsed,
			Leaf:      cfg.Sidebar.Markers.Leaf,
		}).
		Graphics(graphics).
		GraphicsSet(set).
		GraphicsStyle(tcell.StyleDefault.Foreground(tcell.GetColor(theme.GraphicsColor))).
		Keybind(treeKeybind(kbs)).
		Focused(focused).
		OnChange(onChange).
		OnSelect(onSelect)
	v := viewport.New(t, scrollState).
		Target(t.Target).
		ContentWidth(t.RowsWidth).
		OnChildMsg(func(s *viewport.ScrollState, msg tview.Msg) tview.Msg { return ScrollMsg{ScrollState: *s, Msg: msg} }).
		Axes(layout.Axes{Width: true, Height: true}).
		Keybind(scrollKeybind(kbs)).
		Focused(focused).
		OnChange(func(s *viewport.ScrollState) tview.Msg { return ScrollMsg{ScrollState: *s} })
	return ui.Box(v, cfg, focused)
}
