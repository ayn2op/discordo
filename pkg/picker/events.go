package picker

import (
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type SelectedEvent struct {
	tcell.EventTime
	Item
}

func newSelectedEvent(item Item) *SelectedEvent {
	return &SelectedEvent{Item: item}
}

func (m *Model) _select() tview.Command {
	index := m.list.Cursor()
	if index >= 0 && index < len(m.filtered) {
		item := m.filtered[index]
		return func() tview.Event {
			return newSelectedEvent(item)
		}
	}
	return nil
}

type CancelEvent struct{ tcell.EventTime }

func cancel() tview.Command {
	return func() tview.Event {
		return &CancelEvent{}
	}
}
