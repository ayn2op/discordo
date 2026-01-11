package chat

import (
	"context"

	tea "charm.land/bubbletea/v2"
	"github.com/diamondburned/arikawa/v3/gateway"
)

type (
	EventMsg gateway.Event
	ErrMsg   error
)

func (m Model) openState() tea.Msg {
	if err := m.state.Open(context.TODO()); err != nil {
		return ErrMsg(err)
	}
	return nil
}

func (m Model) listen() tea.Msg {
	select {
	case err := <-m.errs:
		return ErrMsg(err)
	case event := <-m.events:
		return EventMsg(event)
	}
}
