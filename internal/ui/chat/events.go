package chat

import "github.com/gdamore/tcell/v3"

type LogoutEvent struct{ tcell.EventTime }

func NewLogoutEvent() *LogoutEvent {
	event := &LogoutEvent{}
	event.SetEventNow()
	return event
}

type QuitEvent struct{ tcell.EventTime }

func NewQuitEvent() *QuitEvent {
	event := &QuitEvent{}
	event.SetEventNow()
	return event
}
