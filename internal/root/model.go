package root

import (
	"github.com/ayn2op/discordo/internal/chat"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/ayn2op/discordo/pkg/tea"
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
	case tokenMsg:
		m.model = chat.NewModel()
		return m, m.model.Init()
	case loginMsg:
		m.model = login.NewModel(m.cfg)
		return m, m.model.Init()

	case tea.KeyMsg:
		switch msg.Name() {
		case m.cfg.Keybinds.Quit:
			return m, tea.Quit
		}
	}

	if m.model == nil {
		return m, nil
	}

	var cmd tea.Cmd
	m.model, cmd = m.model.Update(msg)
	return m, cmd
}

func (m Model) View(frame *tea.Frame, area tea.Rect) {
	if m.model != nil {
		m.model.View(frame, area)
	}
}
