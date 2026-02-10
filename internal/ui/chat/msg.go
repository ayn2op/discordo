package chat

import (
	"context"

	tea "charm.land/bubbletea/v2"
)

func (m Model) connect() tea.Cmd {
	return func() tea.Msg {
		m.state.AddHandler(m.events)
		return m.state.Open(context.Background())
	}
}

func (m Model) listen() tea.Cmd {
	return func() tea.Msg {
		return <-m.events
	}
}

type LogoutMsg struct{}

func (m Model) logout() tea.Cmd {
	return func() tea.Msg {
		if err := m.state.Close(); err != nil {
			return err
		}
		return LogoutMsg{}
	}
}
