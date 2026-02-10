package root

import (
	"charm.land/bubbles/v2/help"
	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	qrModel "github.com/ayn2op/discordo/internal/ui/login/qr"
	tokenModel "github.com/ayn2op/discordo/internal/ui/login/token"
)

const tokenEnvVarKey = "DISCORDO_TOKEN"

type Model struct {
	cfg *config.Config

	inner      tea.Model
	help       help.Model
	windowSize tea.WindowSizeMsg
}

func NewModel(cfg *config.Config) Model {
	return Model{
		cfg: cfg,

		help: help.New(),
	}
}

var _ tea.Model = Model{}

func (m Model) Init() tea.Cmd {
	return nil
}

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.windowSize = msg
		m.help.SetWidth(msg.Width)

	case tea.KeyMsg:
		k := msg.Key()
		switch {
		case key.Matches(k, m.cfg.Keybinds.Help.Binding):
			m.help.ShowAll = !m.help.ShowAll
		case key.Matches(k, m.cfg.Keybinds.Suspend.Binding):
			return m, tea.Suspend
		case key.Matches(k, m.cfg.Keybinds.Quit.Binding):
			return m, tea.Quit
		}

	case tea.EnvMsg:
		if token := msg.Getenv(tokenEnvVarKey); token != "" {
			return m, tokenCmd(token)
		}
		// If the token is not found in the environment variable, retrieve the token from keyring.
		return m, getToken()

	case loginMsg:
		return m.activateModel(login.NewModel())
	case chat.LogoutMsg:
		m, cmd := m.activateModel(login.NewModel())
		return m, tea.Batch(deleteToken(), cmd)

	case tokenMsg:
		return m.activateModel(chat.NewModel(string(msg), m.cfg))
	case tokenModel.TokenMsg:
		token := string(msg)
		m, cmd := m.activateModel(chat.NewModel(token, m.cfg))
		return m, tea.Batch(setToken(token), cmd)
	case qrModel.TokenMsg:
		token := string(msg)
		m, cmd := m.activateModel(chat.NewModel(token, m.cfg))
		return m, tea.Batch(setToken(token), cmd)
	}

	var cmd tea.Cmd
	if m.inner != nil {
		m.inner, cmd = m.inner.Update(msg)
	}
	return m, cmd
}

func (m Model) View() tea.View {
	view := tea.NewView("loading")
	if m.inner != nil {
		view = m.inner.View()
	}

	helpKeyMap := help.KeyMap(m)
	if innerKeyMap, ok := m.inner.(help.KeyMap); ok {
		helpKeyMap = compositeKeyMap{
			inner: innerKeyMap,
			base:  helpKeyMap,
		}
	}

	helpView := m.help.View(helpKeyMap)
	helpH := lipgloss.Height(helpView)

	bodyH := max(m.windowSize.Height-helpH, 0)
	// PlaceVertical doesn't clip when content is taller than the container.
	// Clamp first so the help footer doesn't get pushed below the viewport.
	body := lipgloss.NewStyle().MaxHeight(bodyH).Render(view.Content)
	body = lipgloss.PlaceVertical(bodyH, lipgloss.Top, body)

	view.Content = lipgloss.JoinVertical(lipgloss.Top, body, helpView)
	view.AltScreen = true
	return view
}

func (m Model) activateModel(inner tea.Model) (tea.Model, tea.Cmd) {
	m.inner = inner
	initCmd := m.inner.Init()
	var updateCmd tea.Cmd
	m.inner, updateCmd = m.inner.Update(m.windowSize)
	return m, tea.Sequence(initCmd, updateCmd)
}

var _ help.KeyMap = Model{}

func (m Model) ShortHelp() []key.Binding {
	cfg := m.cfg.Keybinds
	return []key.Binding{cfg.Help.Binding, cfg.Quit.Binding}
}

func (m Model) FullHelp() [][]key.Binding {
	cfg := m.cfg.Keybinds
	return [][]key.Binding{
		{cfg.Help.Binding, cfg.Suspend.Binding, cfg.Quit.Binding},
	}
}

type compositeKeyMap struct {
	inner help.KeyMap
	base  help.KeyMap
}

func (k compositeKeyMap) ShortHelp() []key.Binding {
	short := k.inner.ShortHelp()
	short = append(short, k.base.ShortHelp()...)
	return short
}

func (k compositeKeyMap) FullHelp() [][]key.Binding {
	full := k.inner.FullHelp()
	full = append(full, k.base.FullHelp()...)
	return full
}
