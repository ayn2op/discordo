package root

import (
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/chat"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/login"
)

type Model struct {
	model tea.Model
}

func NewModel() Model {
	return Model{}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return getToken
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "esc" {
			return m, tea.Quit
		}

	case tokenMsg:
		if msg.err != nil {
			m.model = login.NewModel()
		} else {
			m.model = chat.NewModel()
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
	view.WindowTitle = consts.Name

	if m.model != nil {
		view = m.model.View()
	}

	view.AltScreen = true
	return view
}
