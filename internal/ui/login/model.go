package login

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"github.com/ayn2op/discordo/internal/ui/login/qr"
	"github.com/ayn2op/discordo/internal/ui/login/token"
	"github.com/ayn2op/discordo/pkg/tabs"
)

type Model struct {
	tabs tabs.Model
}

func NewModel() Model {
	return Model{
		tabs: tabs.NewModel([]tabs.Tab{token.NewModel(), qr.NewModel()})}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return m.tabs.Init()
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.tabs, cmd = m.tabs.Update(msg)
	return m, cmd
}

func (m Model) View() tea.View {
	return m.tabs.View()
}

var _ help.KeyMap = Model{}

func (m Model) FullHelp() [][]key.Binding {
	return m.tabs.FullHelp()
}

func (m Model) ShortHelp() []key.Binding {
	return m.tabs.ShortHelp()
}
