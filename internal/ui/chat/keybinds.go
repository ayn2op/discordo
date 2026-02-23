package chat

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = (*View)(nil)

func (v *View) ShortHelp() []keybind.Keybind {
	short := make([]keybind.Keybind, 0, 16)
	if active := v.activeKeyMap(); active != nil {
		short = append(short, active.ShortHelp()...)
	}
	short = append(short, v.baseShortHelp()...)
	return short
}

func (v *View) FullHelp() [][]keybind.Keybind {
	full := make([][]keybind.Keybind, 0, 8)
	if active := v.activeKeyMap(); active != nil {
		full = append(full, active.FullHelp()...)
	}
	full = append(full, v.baseFullHelp()...)
	return full
}

func (v *View) activeKeyMap() help.KeyMap {
	if v.GetVisible(channelsPickerLayerName) {
		return v.channelsPicker
	}

	if v.app == nil {
		return nil
	}

	switch v.app.GetFocus() {
	case v.guildsTree:
		return v.guildsTree
	case v.messagesList:
		return v.messagesList
	case v.messageInput:
		return v.messageInput
	default:
		return nil
	}
}

func (v *View) baseShortHelp() []keybind.Keybind {
	cfg := v.cfg.Keybinds
	short := []keybind.Keybind{cfg.FocusGuildsTree.Keybind, cfg.FocusMessagesList.Keybind}
	if !v.messageInput.GetDisabled() {
		short = append(short, cfg.FocusMessageInput.Keybind)
	}
	short = append(short, cfg.ToggleGuildsTree.Keybind, cfg.ToggleChannelsPicker.Keybind, cfg.ToggleHelp.Keybind)
	return short
}

func (v *View) baseFullHelp() [][]keybind.Keybind {
	cfg := v.cfg.Keybinds
	focus := []keybind.Keybind{cfg.FocusGuildsTree.Keybind, cfg.FocusMessagesList.Keybind}
	if !v.messageInput.GetDisabled() {
		focus = append(focus, cfg.FocusMessageInput.Keybind)
	}
	return [][]keybind.Keybind{
		focus,
		{cfg.FocusPrevious.Keybind, cfg.FocusNext.Keybind},
		{cfg.ToggleGuildsTree.Keybind, cfg.ToggleChannelsPicker.Keybind},
		{cfg.ToggleHelp.Keybind, cfg.Suspend.Keybind, cfg.Logout.Keybind, cfg.Quit.Keybind},
	}
}
