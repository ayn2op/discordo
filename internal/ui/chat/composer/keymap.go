package composer

import (
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []keybind.Keybind {
	if m.mentionsListVisible() {
		cfg := m.cfg.Keybinds.MentionsList
		ccfg := m.cfg.Keybinds.Composer
		short := []keybind.Keybind{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, ccfg.TabComplete.Keybind, ccfg.Cancel.Keybind}
		if m.canAttachFiles() {
			short = append(short, ccfg.OpenFilePicker.Keybind)
		}
		return short
	}

	cfg := m.cfg.Keybinds.Composer
	short := []keybind.Keybind{cfg.Send.Keybind, cfg.Newline.Keybind, cfg.Cancel.Keybind, cfg.Paste.Keybind, cfg.OpenEditor.Keybind}
	if m.canEditLastMessage() {
		short = append(short, cfg.EditLast.Keybind)
	}
	if m.sendMessageData.Reference != nil {
		short = append(short, cfg.ToggleReplyMention.Keybind)
	}
	if m.canAttachFiles() {
		short = append(short, cfg.OpenFilePicker.Keybind)
	}
	return short
}

func (m Model) FullHelp() [][]keybind.Keybind {
	if m.mentionsListVisible() {
		mcfg := m.cfg.Keybinds.MentionsList
		ccfg := m.cfg.Keybinds.Composer
		return [][]keybind.Keybind{
			{mcfg.SelectUp.Keybind, mcfg.SelectDown.Keybind, mcfg.SelectTop.Keybind, mcfg.SelectBottom.Keybind},
			{ccfg.TabComplete.Keybind, ccfg.Cancel.Keybind},
		}
	}

	cfg := m.cfg.Keybinds.Composer
	openEditor := []keybind.Keybind{cfg.Paste.Keybind, cfg.OpenEditor.Keybind}

	if m.canAttachFiles() {
		openEditor = append(openEditor, cfg.OpenFilePicker.Keybind)
	}

	compose := []keybind.Keybind{cfg.Send.Keybind, cfg.Newline.Keybind, cfg.Cancel.Keybind, cfg.Undo.Keybind}
	compose = append(compose, cfg.EditLast.Keybind)
	if m.sendMessageData.Reference != nil {
		compose = append(compose, cfg.ToggleReplyMention.Keybind)
	}

	return [][]keybind.Keybind{
		compose,
		openEditor,
	}
}
