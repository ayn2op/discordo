package login

import "github.com/gdamore/tcell/v3"

type TokenEvent struct {
	tcell.EventTime
	Token string
}

func NewTokenEvent(token string) *TokenEvent {
	event := &TokenEvent{Token: token}
	event.SetEventNow()
	return event
}
