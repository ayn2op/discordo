package login

import (
	"log/slog"

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
	passwordTab tab = iota
	qrTab
	tokenTab
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
	return Model{
		cfg:      cfg,
		password: password.NewModel(),
		qr:       qr.NewModel(),
		token:    token.NewModel(),
	}
}

// Init initializes the active tab.
func (m Model) Init() tview.Cmd {
	switch m.active {
	case passwordTab:
		return m.password.Init()
	case qrTab:
		return m.qr.Init()
	case tokenTab:
		return m.token.Init()
	}
	return nil
}

// selectTabMsg switches to the tab at an index.
type selectTabMsg int

// View shows the tabs in a box.
func (m Model) View() tview.Element {
	var content tview.Element
	switch m.active {
	case passwordTab:
		content = m.password.View()
	case qrTab:
		content = m.qr.View()
	case tokenTab:
		content = m.token.View()
	}
	t := tabs.New(m.password.Label(), m.qr.Label(), m.token.Label()).
		Active(int(m.active)).
		Content(content).
		Keybind(tabAction).
		OnSelect(func(i int) tview.Msg { return selectTabMsg(i) })
	return ui.Box(t, &m.cfg.Theme, false)
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
	case copyErrorMsg:
		return m, setClipboard(string(msg))
	}

	var cmd tview.Cmd
	switch m.active {
	case passwordTab:
		m.password, cmd = m.password.Update(msg)
	case qrTab:
		m.qr, cmd = m.qr.Update(msg)
	case tokenTab:
		m.token, cmd = m.token.Update(msg)
	}
	return m, tview.Batch(errCmd, cmd)
}

func showErrorDialog(err error) tview.Cmd {
	slog.Error("failed to login", "err", err)
	message := err.Error()
	return ui.ShowModal(message,
		ui.ModalButton{Label: "Copy", Result: copyErrorMsg(message), KeepOpen: true},
		ui.ModalButton{Label: "Close"},
	)
}
