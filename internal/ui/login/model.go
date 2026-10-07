package login

import (
	"github.com/ayn2op/tview/tabs"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/login/password"
	"github.com/ayn2op/discordo/internal/ui/login/qr"
	"github.com/ayn2op/discordo/internal/ui/login/token"
	"github.com/ayn2op/tview"
)

// tab is a login method shown as a tab.
type tab int

const (
	tabPassword tab = iota
	tabQR
	tabToken
	tabCount
)

// Model shows the login methods as tabs inside a bordered box.
type Model struct {
	cfg      *config.Config
	password password.Model
	qr       qr.Model
	token    token.Model
	active   tab
}

var _ tview.Model[Model] = Model{}

func NewModel(cfg *config.Config) Model {
	return Model{cfg: cfg, qr: qr.NewModel()}
}

// Init initializes the active tab.
func (m Model) Init() tview.Cmd {
	switch m.active {
	case tabPassword:
		return m.password.Init()
	case tabQR:
		return m.qr.Init()
	case tabToken:
		return m.token.Init()
	}
	return nil
}

// View shows the tabs in a box.
func (m Model) View() tview.Widget {
	var content tview.Widget
	switch m.active {
	case tabPassword:
		content = m.password.View()
	case tabQR:
		content = m.qr.View()
	case tabToken:
		content = m.token.View()
	}
	t := m.tabs().
		Active(int(m.active)).
		Content(content).
		Keybind(tabAction).
		OnSelect(func(i int) tview.Msg { return selectTabMsg(i) })
	return ui.Box(t, m.cfg, false)
}

// tabs returns the login methods as tabs set up by the tabs config.
func (m Model) tabs() tabs.Widget {
	cfg := m.cfg.UI.Tabs
	return tabs.New(m.password.Label(), m.qr.Label(), m.token.Label()).
		Alignment(cfg.Alignment.Alignment).
		Wrap(cfg.Wrap).
		Separator(cfg.Separator).
		Padding(cfg.Padding[0], cfg.Padding[1]).
		Arrows(cfg.Arrows[0], cfg.Arrows[1]).
		ClickableArrows(cfg.ClickableArrows)
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	var errCmd tview.Cmd
	switch msg := msg.(type) {
	case selectTabMsg:
		m.active = tab(msg)
		return m, m.Init()
	case error:
		// The active tab still gets the error, so the QR tab can close its connection.
		errCmd = showErrorDialog(msg)
	}

	var cmd tview.Cmd
	switch m.active {
	case tabPassword:
		m.password, cmd = m.password.Update(msg)
	case tabQR:
		m.qr, cmd = m.qr.Update(msg)
	case tabToken:
		m.token, cmd = m.token.Update(msg)
	}
	return m, tview.Batch(errCmd, cmd)
}
