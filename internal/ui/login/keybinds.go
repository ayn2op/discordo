package login

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = (*Model)(nil)

func (m *Model) ShortHelp() []keybind.Keybind {
	return m.tabs.ShortHelp()
}

func (m *Model) FullHelp() [][]keybind.Keybind {
	return m.tabs.FullHelp()
}
