package composer

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/textarea"
)

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []keybind.Keybind {
	if m.autocomplete.Visible() {
		cfg := m.cfg.Keybinds.Autocomplete
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
	if m.autocomplete.Visible() {
		mcfg := m.cfg.Keybinds.Autocomplete
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

// keybind binds the configured newline and editing keys and textarea's other default keys, leaving the composer's own keys and the global ones to Update.
func (m Model) keybind(key tview.KeyMsg) textarea.Action {
	if m.isBound(key) || keybind.Matches(key, m.cfg.Keybinds.Global()...) {
		return textarea.ActionNone
	}
	if keybind.Matches(key, m.cfg.Keybinds.Composer.Newline.Keybind) {
		return textarea.ActionNewline
	}
	if action := m.editAction(key); action != textarea.ActionNone {
		return action
	}
	if action := textarea.DefaultKeybind(key); action != textarea.ActionNewline {
		return action
	}
	return textarea.ActionNone
}

// editAction returns the action of the editing keybind key matches, or ActionNone.
func (m Model) editAction(key tview.KeyMsg) textarea.Action {
	keybinds := m.cfg.Keybinds.Composer
	for _, bind := range []struct {
		keybind config.Keybind
		action  textarea.Action
	}{
		{keybinds.CursorLeft, textarea.ActionLeft},
		{keybinds.CursorRight, textarea.ActionRight},
		{keybinds.CursorUp, textarea.ActionUp},
		{keybinds.CursorDown, textarea.ActionDown},
		{keybinds.LineStart, textarea.ActionHome},
		{keybinds.LineEnd, textarea.ActionEnd},
		{keybinds.WordLeft, textarea.ActionWordLeft},
		{keybinds.WordRight, textarea.ActionWordRight},
		{keybinds.DeleteBackward, textarea.ActionBackspace},
		{keybinds.DeleteForward, textarea.ActionDelete},
		{keybinds.DeleteWordBackward, textarea.ActionDeleteWordBackward},
		{keybinds.DeleteWordForward, textarea.ActionDeleteWordForward},
		{keybinds.DeleteToSpaceBackward, textarea.ActionDeleteToSpaceBackward},
		{keybinds.DeleteToLineStart, textarea.ActionDeleteToLineStart},
		{keybinds.DeleteToLineEnd, textarea.ActionDeleteToLineEnd},
	} {
		if keybind.Matches(key, bind.keybind.Keybind) {
			return bind.action
		}
	}
	return textarea.ActionNone
}
