package chat

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/http"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/state/store/defaultstore"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/diamondburned/ningen/v3"
)

type Model struct {
	cfg    *config.Config
	state  *ningen.State
	events chan gateway.Event
}

func NewModel(token string, cfg *config.Config) Model {
	identifyProps := http.IdentifyProperties()
	gateway.DefaultIdentity = identifyProps
	gateway.DefaultPresence = &gateway.UpdatePresenceCommand{
		Status: cfg.Status,
	}
	id := gateway.DefaultIdentifier(token)
	id.Compress = false
	id.LargeThreshold = 0
	session := session.NewCustom(id, http.NewClient(token), handler.New())
	return Model{
		cfg:    cfg,
		state:  ningen.FromState(state.NewFromSession(session, defaultstore.New())),
		events: make(chan gateway.Event),
	}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return tea.Batch(m.connect(), m.listen())
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.cfg.Keybinds.Logout.Binding):
			return m, m.logout()
		}
	}
	return m, m.listen()
}

func (m Model) View() tea.View {
	return tea.NewView("chat")
}

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []key.Binding {
	cfg := m.cfg.Keybinds
	return []key.Binding{cfg.FocusGuildsTree.Binding, cfg.FocusMessagesList.Binding, cfg.FocusMessageInput.Binding}
}

func (m Model) FullHelp() [][]key.Binding {
	cfg := m.cfg.Keybinds
	return [][]key.Binding{
		{cfg.FocusGuildsTree.Binding, cfg.FocusMessagesList.Binding, cfg.FocusMessageInput.Binding},
		{cfg.FocusPrevious.Binding, cfg.FocusNext.Binding},
	}
}
