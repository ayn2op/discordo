package root

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/chat"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/login"
)

type Model struct {
	model tea.Model
	cfg   *config.Config
}

func NewModel(cfg *config.Config) Model {
	return Model{cfg: cfg}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return getToken
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case m.cfg.Keys.Quit:
			return m, tea.Quit
		}

	case TokenMsg:
		if msg.err != nil {
			m.model = login.NewModel(m.cfg)
		} else {
			m.model = chat.NewModel(m.cfg, msg.Value)
		}
		return m, m.model.Init()
	}

	var cmd tea.Cmd
	if m.model != nil {
		m.model, cmd = m.model.Update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	view := tea.NewView("Loading...")

	if m.model != nil {
		view = m.model.View()
	}

	if view.WindowTitle != "" {
		view.WindowTitle += " - " + consts.Name
	} else {
		view.WindowTitle = consts.Name
	}

	view.AltScreen = true
	return view
}
