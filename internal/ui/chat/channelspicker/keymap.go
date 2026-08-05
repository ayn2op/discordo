package channelspicker

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = (*Model)(nil)

func (m *Model) ShortHelp() []keybind.Keybind {
	cfg := m.cfg.Keybinds.Picker
	return []keybind.Keybind{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, cfg.Select.Keybind, cfg.Cancel.Keybind}
}

func (m *Model) FullHelp() [][]keybind.Keybind {
	cfg := m.cfg.Keybinds.Picker
	return [][]keybind.Keybind{
		{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, cfg.SelectTop.Keybind, cfg.SelectBottom.Keybind},
		{cfg.Select.Keybind, cfg.Cancel.Keybind},
	}
}
