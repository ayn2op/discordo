package memberstree

import (
	uitree "github.com/ayn2op/discordo/internal/ui/tree"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []keybind.Keybind {
	kbs := m.cfg.Keybinds.MembersTree
	short := uitree.SelectHelp(m.root, m.selectionState.CurrentNode(), kbs)
	if node := m.selectionState.CurrentNode(); node != nil {
		if _, ok := node.Reference().(groupRef); ok {
			desc := "expand"
			if node.Expanded() {
				desc = "collapse"
			}
			short = append(short, kbs.SelectCurrent.WithHelp(kbs.SelectCurrent.Help().Key, desc))
		}
	}
	if m.parent(m.selectionState.CurrentNode()) != nil {
		short = append(short, kbs.CollapseParentNode.Keybind)
	}
	return short
}

func (m Model) FullHelp() [][]keybind.Keybind {
	kbs := m.cfg.Keybinds.MembersTree
	return [][]keybind.Keybind{
		{kbs.SelectUp.Keybind, kbs.SelectDown.Keybind, kbs.SelectTop.Keybind, kbs.SelectBottom.Keybind},
		{kbs.ScrollUp.Keybind, kbs.ScrollDown.Keybind, kbs.ScrollTop.Keybind, kbs.ScrollBottom.Keybind, kbs.ScrollLeft.Keybind, kbs.ScrollRight.Keybind},
		{kbs.SelectCurrent.Keybind, kbs.MoveToParentNode.Keybind, kbs.CollapseParentNode.Keybind, kbs.CollapseAll.Keybind},
		{kbs.YankID.Keybind},
	}
}
