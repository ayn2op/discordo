package main

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v2"
)

func onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case config.Current.Keys.GuildsTree.Focus:
		app.SetFocus(guildsTree)
		return nil
	case config.Current.Keys.MessagesText.Focus:
		app.SetFocus(messagesText)
		return nil
	case config.Current.Keys.MessageInput.Focus:
		app.SetFocus(messageInput)
		return nil
	}

	return event
}
