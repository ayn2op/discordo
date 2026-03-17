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

func (p *Picker) _select() tview.Command {
	index := p.list.Cursor()
	if index >= 0 && index < len(p.filtered) {
		item := p.filtered[index]
		return func() tcell.Event {
			return newSelectedEvent(item)
		}
	}
	return nil
}

type CancelEvent struct{ tcell.EventTime }

func cancel() tview.Command {
	return func() tcell.Event {
		return &CancelEvent{}
	}
}
