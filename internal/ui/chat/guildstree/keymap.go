package guildstree

import (
	"slices"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/tree"
)

var _ help.KeyMap = Model{}

func (m *Model) selectCurrentKeybind() keybind.Keybind {
	selectCurrent := m.cfg.Keybinds.GuildsTree.SelectCurrent.Keybind
	selectHelp := selectCurrent.Help()
	selectDesc := selectHelp.Desc
	if node := m.selectionState.CurrentNode(); node != nil {
		if len(node.Children()) > 0 {
			if node.Expanded() {
				selectDesc = "collapse"
			} else {
				selectDesc = "expand"
			}
		} else {
			switch node.Reference().(type) {
			case discord.GuildID, dmNode:
				selectDesc = "expand"
			}
		}
	}
	return selectCurrent.WithHelp(selectHelp.Key, selectDesc)
}

func (m Model) ShortHelp() []keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree
	shortHelp := []keybind.Keybind{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, m.selectCurrentKeybind()}
	if m.canCollapseParent(m.selectionState.CurrentNode()) {
		shortHelp = append(shortHelp, cfg.CollapseParentNode.Keybind)
	}
	return shortHelp
}

func (m Model) FullHelp() [][]keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree
	selectGroup := []keybind.Keybind{m.selectCurrentKeybind(), cfg.MoveToParentNode.Keybind}
	selectGroup = append(selectGroup, m.collapseKeybinds()...)

	return [][]keybind.Keybind{
		{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, cfg.SelectTop.Keybind, cfg.SelectBottom.Keybind},
		selectGroup,
		{cfg.YankID.Keybind},
	}
}

func (m *Model) collapseKeybinds() []keybind.Keybind {
	cfg := m.cfg.Keybinds.GuildsTree

	var keybinds []keybind.Keybind
	if m.canCollapseParent(m.selectionState.CurrentNode()) {
		keybinds = append(keybinds, cfg.CollapseParentNode.Keybind)
	}
	if m.canCollapseAll() {
		keybinds = append(keybinds, cfg.CollapseAll.Keybind)
	}
	return keybinds
}

func (m *Model) canCollapseParent(node *tree.Node) bool {
	return node != nil && len(m.root.PathTo(node)) >= 3
}

func (m *Model) canCollapseAll() bool {
	return slices.ContainsFunc(m.root.Children(), (*tree.Node).Expanded)
}
