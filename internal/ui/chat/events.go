package chat

import (
	"log/slog"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type LogoutEvent struct{ tcell.EventTime }

func (m *Model) logout() tview.Command {
	return func() tview.Event {
		return &LogoutEvent{}
	}
}

type QuitEvent struct{ tcell.EventTime }

func (m *Model) closeState() tview.Command {
	return func() tview.Event {
		if m.state != nil {
			if err := m.state.Close(); err != nil {
				slog.Error("failed to close the session", "err", err)
				return tcell.NewEventError(err)
			}
		}
		return nil
	}
}

type closeLayerEvent struct {
	tcell.EventTime
	name string
}

func closeLayer(name string) tview.Command {
	return func() tview.Event {
		return &closeLayerEvent{name: name}
	}
}
