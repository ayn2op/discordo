package chat

import (
	"log/slog"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

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

func (v *View) closeState() tcell.Event {
	if err := v.CloseState(); err != nil {
		slog.Error("failed to close the session", "err", err)
		return tcell.NewEventError(err)
	}
	return nil
}

func (v *View) logout() tview.Command {
	return tview.BatchCommand{
		tview.EventCommand(v.closeState),
		tview.EventCommand(func() tcell.Event { return NewLogoutEvent() }),
	}
}
