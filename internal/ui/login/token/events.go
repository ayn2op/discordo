package token

import (
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type TokenEvent struct {
	tcell.EventTime
	Token string
}

func tokenCommand(token string) tview.Command {
	return func() tcell.Event {
		event := &TokenEvent{Token: token}
		event.SetEventNow()
		return event
	}
}
