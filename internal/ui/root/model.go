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
	"github.com/ayn2op/tview/cmd"
	"github.com/ayn2op/tview/column"
	"github.com/ayn2op/tview/dialog"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/inert"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/stack"
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
	// dialog is the dialog shown, or nil when none is open, and dialogFocus its focused button.
	dialog      *ui.DialogMsg
	dialogFocus int
	// resize is the last size of the window, or nil before the first, kept for the models shown later.
	resize tview.ResizeMsg

	cfg *config.Config
}

func NewModel(cfg *config.Config) Model {
	return Model{cfg: cfg, helpVisible: cfg.Help.Enabled}
}

// updateScreen updates the model shown above help with msg.
func (m *Model) updateScreen(msg tview.Msg) tview.Cmd {
	var c tview.Cmd
	switch m.screen {
	case screenLogin:
		m.login, c = m.login.Update(msg)
	case screenChat:
		m.chat, c = m.chat.Update(msg)
	}
	return c
}

// helpView returns the help for the keybinds that currently apply.
func (m Model) helpView() help.Widget {
	cfg, muted := m.cfg, ui.MutedStyle(m.cfg.Palette())
	return help.New(m).
		ShortKeyStyle(muted).
		ShortSeparatorStyle(muted).
		FullKeyStyle(muted).
		FullSeparatorStyle(muted).
		EllipsisStyle(muted).
		CompactModifiers(cfg.Help.CompactModifiers).
		ShortSeparator(cfg.Help.Separator).
		ShowAll(m.helpShowAll)
}

type (
	dialogFocusMsg int
	dialogDoneMsg  int
)

// dialogView returns the view of the open dialog.
func (m Model) dialogView() tview.Widget {
	labels := make([]string, len(m.dialog.Buttons))
	for i, button := range m.dialog.Buttons {
		labels[i] = button.Label
	}
	palette := m.cfg.Palette()
	return dialog.New().
		Text(m.dialog.Text).
		Buttons(labels...).
		ButtonStyle(ui.ButtonStyle(palette, false, false)).
		ActivatedStyle(ui.ButtonStyle(palette, true, false)).
		Focus(m.dialogFocus).
		OnFocus(func(i int) tview.Msg { return dialogFocusMsg(i) }).
		OnDone(func(i int, _ string) tview.Msg { return dialogDoneMsg(i) })
}

func (m *Model) toggleHelp() {
	if m.cfg.Help.Enabled {
		m.helpVisible = !m.helpVisible
	}
}

var _ tview.Model[Model] = Model{}

func (m Model) Init() tview.Cmd {
	var c tview.Cmd
	if token := os.Getenv(tokenEnvVarKey); token != "" {
		c = useToken(token)
	} else {
		c = getToken()
	}
	return cmd.Batch{
		cmd.SetTitle(consts.Name),
		ui.SetColors(m.cfg.Palette().Background.Base.Style),
		initClipboard(),
		c,
	}
}

// Update returns m changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case loginMsg:
		return m, m.showLogin()
	case tokenMsg:
		return m, m.showChat(string(msg))

	case password.TokenMsg:
		return m, cmd.Batch{m.showChat(string(msg)), setToken(string(msg))}
	case qr.TokenMsg:
		return m, cmd.Batch{m.showChat(string(msg)), setToken(string(msg))}
	case token.TokenMsg:
		return m, cmd.Batch{m.showChat(string(msg)), setToken(string(msg))}

	case chat.ClosedMsg:
		return m, cmd.Quit{}
	case chat.LogoutMsg:
		if msg.Err != nil {
			dialog := ui.ErrorDialog("open chat state", msg.Err)
			m.dialog, m.dialogFocus = &dialog, 0
		}
		return m, cmd.Batch{
			m.showLogin(),
			deleteToken(),
		}
	case ui.DialogMsg:
		if m.dialog == nil {
			m.dialog, m.dialogFocus = &msg, 0
		}
		return m, nil
	case dialogFocusMsg:
		m.dialogFocus = int(msg)
		return m, nil
	case dialogDoneMsg:
		return m, m.finishDialog(int(msg))

	case tview.KeyMsg:
		// The dialog takes all input while it is open, so nothing behind it reacts.
		if m.dialog != nil {
			return m, nil
		}

		keybinds := m.cfg.Keybinds
		switch {
		case keybind.Matches(msg, keybinds.ToggleHelp.Keybind):
			m.toggleHelp()
			return m, nil
		case keybind.Matches(msg, keybinds.ToggleFullHelp.Keybind):
			if m.cfg.Help.Enabled {
				m.helpShowAll = !m.helpShowAll
			}
			return m, nil
		case keybind.Matches(msg, keybinds.Suspend.Keybind):
			return m, suspend()
		case keybind.Matches(msg, keybinds.Quit.Keybind):
			return m, ui.ShowDialog(
				"Are you sure you want to quit?",
				ui.DialogButton{Label: "No"},
				ui.DialogButton{Label: "Yes", Cmd: m.quit()},
			)
		}
	case tview.PasteMsg:
		if m.dialog != nil {
			return m, nil
		}
	case tview.ResizeMsg:
		m.resize = msg
	}

	return m, m.updateScreen(msg)
}

// View shows the inner model above help, with the dialog on top of both when one is open.
func (m Model) View() tview.Widget {
	var innerView, helpView tview.Widget
	switch m.screen {
	case screenLogin:
		innerView = m.login.View()
	case screenChat:
		innerView = m.chat.View()
	}
	if m.helpVisible {
		helpView = box.New(m.helpView()).Padding(0, 0, m.cfg.Help.Padding[0], m.cfg.Help.Padding[1]).Height(layout.Shrink)
	}
	// Fill the background with the theme's.
	content := box.New(column.New(innerView, helpView)).Background(m.cfg.Palette().Background.Base.GetBackground())
	if m.dialog == nil {
		return content
	}
	return stack.New(inert.New(content), backdrop.New().Style(ui.MutedStyle(m.cfg.Palette())), m.dialogView())
}
