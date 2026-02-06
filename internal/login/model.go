package login

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/login/password"
	"github.com/ayn2op/discordo/internal/login/qr"
	"github.com/ayn2op/discordo/internal/login/token"
	"github.com/ayn2op/discordo/pkg/tabs"
	"github.com/ayn2op/discordo/pkg/tea"
)

type Model struct {
	tabs tabs.Model
}

func NewModel(cfg *config.Config) Model {
	keybinds := tabs.DefaultKeybinds()
	if cfg != nil {
		if cfg.Keybinds.FocusPrevious != "" {
			keybinds.Previous = cfg.Keybinds.FocusPrevious
		}
		if cfg.Keybinds.FocusNext != "" {
			keybinds.Next = cfg.Keybinds.FocusNext
		}
	}

	tabModel := tabs.NewModel([]tabs.Tab{
		token.NewModel(),
		password.NewModel(),
		qr.NewModel(),
	})
	tabModel.Keybinds = keybinds
	return Model{tabs: tabModel}
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

func (m Model) View(frame *tea.Frame, area tea.Rect) {
	m.tabs.View(frame, area)
}
