package chat

import (
	"log/slog"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type LogoutEvent struct{ tcell.EventTime }

func (v *Model) logout() tview.Command {
	return func() tcell.Event {
		return &LogoutEvent{}
	}
}

type QuitEvent struct{ tcell.EventTime }

func (v *Model) closeState() tview.Command {
	return func() tcell.Event {
		if err := v.CloseState(); err != nil {
			slog.Error("failed to close the session", "err", err)
			return tcell.NewEventError(err)
		}
		return nil
	}
}

type closeLayerEvent struct {
	tcell.EventTime
	name string
}

func closeLayer(name string) tview.Command {
	return func() tcell.Event {
		return &closeLayerEvent{name: name}
	}
}
