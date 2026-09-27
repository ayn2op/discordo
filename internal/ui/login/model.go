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
type tab interface {
	tview.Model
	Label() string
}

// Model shows the login methods as tabs inside a bordered box.
type Model struct {
	cfg      *config.Config
	tabs     []tab
	active   int
	keybinds tabs.Keybinds
}

func NewModel(cfg *config.Config) *Model {
	return &Model{
		cfg:      cfg,
		tabs:     []tab{password.NewModel(), qr.NewModel(), token.NewModel()},
		keybinds: tabs.DefaultKeybinds(),
	}
}

// Init initializes the active tab.
func (m *Model) Init() tview.Cmd {
	return m.tabs[m.active].Init()
}

// selectTabMsg switches to the tab at an index.
type selectTabMsg int

// View shows the tabs in a box.
func (m *Model) View() tview.Element {
	labels := make([]string, len(m.tabs))
	for i, tab := range m.tabs {
		labels[i] = tab.Label()
	}
	t := tabs.New(labels...).
		Active(m.active).
		Content(m.tabs[m.active].View()).
		Keybinds(m.keybinds).
		OnSelect(func(i int) tview.Msg { return selectTabMsg(i) })
	return ui.Box(t, &m.cfg.Theme, false)
}

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case selectTabMsg:
		m.active = int(msg)
		return m.tabs[m.active].Init()
	case error:
		return showErrorDialog(msg)
	case copyErrorMsg:
		return setClipboard(string(msg))
	}
	return m.tabs[m.active].Update(msg)
}

func showErrorDialog(err error) tview.Cmd {
	slog.Error("failed to login", "err", err)
	message := err.Error()
	return ui.ShowModal(message,
		ui.ModalButton{Label: "Copy", Result: copyErrorMsg(message), KeepOpen: true},
		ui.ModalButton{Label: "Close"},
	)
}
