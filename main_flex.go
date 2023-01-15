package main

import (
	"log"

	"github.com/gdamore/tcell/v2"
)

func onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	log.Println(event.Name())
	switch event.Name() {
	case cfg.Keys.GuildsTree.Focus:
		app.SetFocus(guildsTree)
		return nil
	case cfg.Keys.MessagesText.Focus:
		app.SetFocus(messagesText)
		return nil
	case cfg.Keys.MessageInput.Focus:
		app.SetFocus(messageInput)
		return nil
	}

	return event
}
