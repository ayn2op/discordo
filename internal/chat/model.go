package chat

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/chat/guilds"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
)

type Model struct {
	guilds guilds.Model

	cfg   *config.Config
	state *ningen.State

	events chan gateway.Event
	errs   chan error
}

func NewModel(cfg *config.Config, token string) Model {
	return Model{
		guilds: guilds.NewModel(),

		cfg:   cfg,
		state: ningen.New(token),

		events: make(chan gateway.Event),
		errs:   make(chan error),
	}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	ws.WSError = func(err error) { m.errs <- err }
	m.state.AddHandler(m.events)
	return tea.Batch(m.listen, m.openState, m.guilds.Init())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.guilds, cmd = m.guilds.Update(msg)
	return m, tea.Batch(m.listen, cmd)
}

func (m Model) View() tea.View {
	view := m.guilds.View()
	view.WindowTitle = "Chat"
	return view
}
