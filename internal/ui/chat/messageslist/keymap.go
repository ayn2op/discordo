package messageslist

import (
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
)

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []keybind.Keybind {
	cfg := m.cfg.Keybinds.MessagesList
	help := []keybind.Keybind{
		cfg.SelectUp.Keybind,
		cfg.SelectDown.Keybind,
		cfg.Cancel.Keybind,
	}

	if selectedMessage, ok := m.selectedMessage(); ok {
		if !ui.IsMe(m.state, selectedMessage.Author.ID) {
			help = append(help, cfg.Reply.Keybind)
		}
		if len(selectedMessage.Attachments) != 0 || len(messageURLs(*selectedMessage)) != 0 {
			help = append(help, cfg.Open.Keybind)
		}
	}

	return help
}

func (m Model) FullHelp() [][]keybind.Keybind {
	cfg := m.cfg.Keybinds.MessagesList
	var actions, manage, attachments []keybind.Keybind
	if message, ok := m.selectedMessage(); ok {
		mine := ui.IsMe(m.state, message.Author.ID)
		if !mine {
			actions = append(actions, cfg.Reply.Keybind, cfg.ReplyMention.Keybind)
		}
		if message.ReferencedMessage != nil {
			actions = append(actions, cfg.SelectReply.Keybind)
		}
		if mine {
			manage = append(manage, cfg.Edit.Keybind)
		}
		if m.canDeleteMessage(*message) {
			manage = append(manage, cfg.DeleteConfirm.Keybind)
			if len(cfg.Delete.Keys()) != 0 {
				manage = append(manage, cfg.Delete.Keybind)
			}
		}
		if len(message.Attachments) != 0 || len(messageURLs(*message)) != 0 {
			attachments = append(attachments, cfg.Open.Keybind, cfg.OpenInBrowser.Keybind)
		}
		if len(message.Attachments) != 0 {
			attachments = append(attachments, cfg.Download.Keybind)
		}
	}
	actions = append(actions, cfg.Cancel.Keybind)

	return [][]keybind.Keybind{
		{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, cfg.SelectTop.Keybind, cfg.SelectBottom.Keybind},
		{cfg.ScrollUp.Keybind, cfg.ScrollDown.Keybind, cfg.ScrollTop.Keybind, cfg.ScrollBottom.Keybind},
		actions,
		manage,
		attachments,
		{cfg.YankContent.Keybind, cfg.YankURL.Keybind, cfg.YankID.Keybind},
	}
}
