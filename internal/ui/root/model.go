package root

import (
	"github.com/ayn2op/tview/layout"
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
	screenNone screen = iota
	screenLogin
	screenChat
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
	// resize is the last size of the window, or nil before the first, kept for the models shown later.
	resize tview.ResizeMsg

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
		if m.modal == nil {
			m.modal, m.dialogFocus = &msg, 0
		}
		return nil
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
	case tview.ResizeMsg:
		m.resize = msg
	}

	return m.updateScreen(msg)
}

// updateScreen updates the model shown above help with msg.
func (m *Model) updateScreen(msg tview.Msg) tview.Cmd {
	var cmd tview.Cmd
	switch m.screen {
	case screenLogin:
		m.login, cmd = m.login.Update(msg)
	case screenChat:
		m.chat, cmd = m.chat.Update(msg)
	}
	return cmd
}

// helpHeight returns the rows the help takes at an unlimited width, as the width of the screen is not known in View.
func (m Model) helpHeight() int {
	return m.helpView().Layout(layout.Limits{Infinite: layout.Axes{Height: true}}).Height
}

// View shows the inner model above help, with the modal dialog on top of both when one is open.
func (m Model) View() tview.Widget {
	var innerView, helpView tview.Widget
	switch m.screen {
	case screenLogin:
		innerView = m.login.View()
	case screenChat:
		innerView = m.chat.View()
	}
	if m.helpVisible {
		padded := box.New(m.helpView()).Padding(0, 0, m.cfg.Help.Padding[0], m.cfg.Help.Padding[1])
		helpView = column.New(padded).Height(layout.Fixed(max(m.helpHeight(), 1)))
	}
	content := column.New(innerView, helpView)
	if m.modal == nil {
		return content
	}
	return stack.New(inert.New(content), backdrop.New().Style(m.cfg.Theme.Dialog.BackgroundStyle.Style), m.dialogView())
}

// helpView returns the help for the keybinds that currently apply.
func (m Model) helpView() help.Widget {
	cfg := m.cfg
	return help.New(m).
		ShortKeyStyle(cfg.Theme.Help.ShortKeyStyle.Style).
		ShortDescStyle(cfg.Theme.Help.ShortDescStyle.Style).
		FullKeyStyle(cfg.Theme.Help.FullKeyStyle.Style).
		FullDescStyle(cfg.Theme.Help.FullDescStyle.Style).
		CompactModifiers(cfg.Help.CompactModifiers).
		ShortSeparator(cfg.Help.Separator).
		ShowAll(m.helpShowAll)
}

type (
	dialogFocusMsg int
	dialogDoneMsg  int
)

// dialogView returns the dialog for the open modal request.
func (m Model) dialogView() tview.Widget {
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
