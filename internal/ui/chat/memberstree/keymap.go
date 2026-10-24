package memberstree

import (
	uitree "github.com/ayn2op/discordo/internal/ui/tree"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []keybind.Keybind {
	keybinds := m.cfg.Keybinds.MembersTree
	short := uitree.SelectHelp(m.root, m.current, keybinds)
	if isGroup(m.current) {
		desc := "expand"
		if m.current.Expanded() {
			desc = "collapse"
		}

		short = append(short, keybinds.SelectCurrent.WithHelp(keybinds.SelectCurrent.Help().Key, desc))
	}

	if m.parent(m.current) != nil {
		short = append(short, keybinds.CollapseParentNode.Keybind)
	}

	return short
}

func (m Model) FullHelp() [][]keybind.Keybind {
	keybinds := m.cfg.Keybinds.MembersTree
	return [][]keybind.Keybind{
		{keybinds.SelectUp.Keybind, keybinds.SelectDown.Keybind, keybinds.SelectTop.Keybind, keybinds.SelectBottom.Keybind},
		{keybinds.ScrollUp.Keybind, keybinds.ScrollDown.Keybind, keybinds.ScrollTop.Keybind, keybinds.ScrollBottom.Keybind, keybinds.ScrollLeft.Keybind, keybinds.ScrollRight.Keybind},
		{keybinds.SelectCurrent.Keybind, keybinds.MoveToParentNode.Keybind, keybinds.CollapseParentNode.Keybind, keybinds.CollapseAll.Keybind},
		{keybinds.YankID.Keybind},
	}
}
