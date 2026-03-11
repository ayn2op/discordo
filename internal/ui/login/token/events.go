package token

import (
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type TokenEvent struct {
	tcell.EventTime
	Token string
}

func newTokenEvent(token string) *TokenEvent {
	event := &TokenEvent{Token: token}
	event.SetEventNow()
	return event
}

func tokenCommand(token string) tview.Command {
	return tview.EventCommand(func() tcell.Event {
		return newTokenEvent(token)
	})
}
