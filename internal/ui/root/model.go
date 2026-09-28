package root

import (
	"os"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	"github.com/ayn2op/discordo/internal/ui/login/password"
	"github.com/ayn2op/discordo/internal/ui/login/qr"
	"github.com/ayn2op/discordo/internal/ui/login/token"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/backdrop"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/column"
	"github.com/ayn2op/tview/dialog"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/inert"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/stack"
	"github.com/gdamore/tcell/v3"
)

const tokenEnvVarKey = "DISCORDO_TOKEN"

// screen is what is shown above help.
type screen int

const (
	noScreen screen = iota
	loginScreen
	chatScreen
)

type Model struct {
	// screen is the model shown above help: login, chat, or neither before the token is known.
	screen screen
	login  login.Model
	chat   chat.Model
	// helpVisible reports whether help is shown below the inner model, and helpShowAll whether it shows every keybind.
	helpVisible, helpShowAll bool
	// modal is the request shown in a dialog, or nil when none is open, and dialogFocus its focused button.
	modal       *ui.ModalMsg
	dialogFocus int

	cfg *config.Config
}

func NewModel(cfg *config.Config) Model {
	return Model{cfg: cfg, helpVisible: cfg.Help.Enabled}
}

var _ tview.Model[Model] = Model{}

func (m Model) Init() tview.Cmd {
	var cmd tview.Cmd
	if token := os.Getenv(tokenEnvVarKey); token != "" {
		cmd = useToken(token)
	} else {
		cmd = getToken()
	}
	return tview.Batch(
		tview.SetTitle(consts.Name),
		initClipboard(),
		cmd,
	)
}

func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	cmd := m.update(msg)
	return m, cmd
}

// update changes m in response to msg and returns a command to run, or nil.
func (m *Model) update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case loginMsg:
		return m.showLogin()
	case tokenMsg:
		return m.showChat(string(msg))

	case password.TokenMsg:
		return tview.Batch(m.showChat(string(msg)), setToken(string(msg)))
	case qr.TokenMsg:
		return tview.Batch(m.showChat(string(msg)), setToken(string(msg)))
	case token.TokenMsg:
		return tview.Batch(m.showChat(string(msg)), setToken(string(msg)))

	case chat.LogoutMsg:
		return tview.Batch(
			m.showLogin(),
			deleteToken(),
		)
	case ui.ModalMsg:
		return m.showModal(msg)
	case dialogFocusMsg:
		m.dialogFocus = int(msg)
		return nil
	case dialogDoneMsg:
		return m.finishModal(int(msg))

	case tview.KeyMsg:
		// The dialog takes all input while it is open, so nothing behind it reacts.
		if m.modal != nil {
			return nil
		}
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleHelp.Keybind):
			m.toggleHelp()
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleFullHelp.Keybind):
			if m.cfg.Help.Enabled {
				m.helpShowAll = !m.helpShowAll
			}
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.Suspend.Keybind):
			return suspend()
		case keybind.Matches(msg, m.cfg.Keybinds.Quit.Keybind):
			return tview.Batch(m.updateScreen(chat.QuitMsg{}), tview.Quit())
		}
	case tview.MouseMsg, tview.PasteMsg:
		if m.modal != nil {
			return nil
		}
	}

	return m.updateScreen(msg)
}

// updateScreen updates the model shown above help with msg.
func (m *Model) updateScreen(msg tview.Msg) tview.Cmd {
	var cmd tview.Cmd
	switch m.screen {
	case loginScreen:
		m.login, cmd = m.login.Update(msg)
	case chatScreen:
		m.chat, cmd = m.chat.Update(msg)
	}
	return cmd
}

// View shows the inner model above help, with the modal dialog on top of both when one is open.
func (m Model) View() tview.Element {
	var innerView, helpView tview.Element
	switch m.screen {
	case loginScreen:
		innerView = m.login.View()
	case chatScreen:
		innerView = m.chat.View()
	}
	if m.helpVisible {
		padded := box.New(m.helpView()).Padding(0, 0, m.cfg.Help.Padding[0], m.cfg.Help.Padding[1])
		helpView = column.New(padded).Height(tview.Fixed(max(m.helpView().Rows(0), 1)))
	}
	content := column.New(innerView, helpView)
	if m.modal == nil {
		return content
	}
	return stack.New(inert.New(content), backdrop.New().Style(m.cfg.Theme.Dialog.BackgroundStyle.Style), m.dialogView())
}

func (m *Model) showModal(request ui.ModalMsg) tview.Cmd {
	if m.modal != nil {
		return nil
	}
	m.modal, m.dialogFocus = &request, 0
	return nil
}

func (m *Model) showLogin() tview.Cmd {
	m.screen, m.login, m.chat = loginScreen, login.NewModel(m.cfg), chat.Model{}
	return m.show(m.login.Init())
}

func (m *Model) showChat(token string) tview.Cmd {
	m.screen, m.chat, m.login = chatScreen, chat.NewModel(m.cfg, token), login.Model{}
	return m.show(m.chat.Init())
}

// show closes any open modal after the shown model changed and returns what it needs done, with init its first command.
func (m *Model) show(init tview.Cmd) tview.Cmd {
	m.modal = nil
	return tview.Batch(tview.SetTitle(consts.Name), init)
}

// helpView returns the help for the keybinds that currently apply.
func (m Model) helpView() help.Widget {
	cfg := m.cfg
	styles := help.DefaultStyles()
	styles.ShortKey = cfg.Theme.Help.ShortKeyStyle.Style
	styles.ShortDesc = cfg.Theme.Help.ShortDescStyle.Style
	styles.FullKey = cfg.Theme.Help.FullKeyStyle.Style
	styles.FullDesc = cfg.Theme.Help.FullDescStyle.Style
	return help.New(m).
		Styles(styles).
		CompactModifiers(cfg.Help.CompactModifiers).
		ShortSeparator(cfg.Help.Separator).
		ShowAll(m.helpShowAll)
}

type (
	dialogFocusMsg int
	dialogDoneMsg  int
)

// dialogView returns the dialog for the open modal request.
func (m Model) dialogView() tview.Element {
	labels := make([]string, len(m.modal.Buttons))
	for i, button := range m.modal.Buttons {
		labels[i] = button.Label
	}
	style := m.cfg.Theme.Dialog.Style.Style
	d := dialog.New().
		Text(m.modal.Text).
		Buttons(labels...).
		Focus(m.dialogFocus).
		ButtonStyle(style).
		ActivatedStyle(style.Reverse(true)).
		OnFocus(func(i int) tview.Msg { return dialogFocusMsg(i) }).
		OnDone(func(i int, _ string) tview.Msg { return dialogDoneMsg(i) })
	if bg := style.GetBackground(); bg != tcell.ColorDefault {
		d = d.Background(bg)
	}
	if fg := style.GetForeground(); fg != tcell.ColorDefault {
		d = d.TextColor(fg)
	}
	return d
}

func (m *Model) toggleHelp() {
	if m.cfg.Help.Enabled {
		m.helpVisible = !m.helpVisible
	}
}

func (m *Model) finishModal(index int) tview.Cmd {
	state := m.modal
	if state == nil {
		return nil
	}

	var result tview.Msg
	if index >= 0 && index < len(state.Buttons) {
		button := state.Buttons[index]
		if button.KeepOpen {
			return func() tview.Msg { return button.Result }
		}
		result = button.Result
	}

	m.modal = nil
	if result == nil {
		return nil
	}
	return func() tview.Msg { return result }
}
