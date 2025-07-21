package home

import (
	"context"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diamondburned/ningen/v3"
)

type Model struct {
	state  *ningen.State
	events chan tea.Msg
}

func NewModel(token string) Model {
	state := ningen.New(token)
	events := make(chan tea.Msg)

	state.AddHandler(func(event any) {
		events <- event

	})

	return Model{
		state:  state,
		events: events,
	}
}

func (m Model) Init() tea.Cmd {
	go m.state.Open(context.TODO())
	return m.listen
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case *ningen.ConnectedEvent:
		_ = msg
	}

	return m, m.listen
}

func (m Model) View() string {
	return "Home"
}

func (m Model) listen() tea.Msg {
	return <-m.events
}
