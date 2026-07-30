package root

import (
	"os"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	"github.com/ayn2op/discordo/internal/ui/login/qr"
	"github.com/ayn2op/discordo/internal/ui/login/token"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/flex"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/layers"
	"github.com/ayn2op/tview/modal"
	"github.com/gdamore/tcell/v3"
)

const (
	tokenEnvVarKey   = "DISCORDO_TOKEN"
	contentLayerName = "content"
	modalLayerName   = "modal"
)

type Model struct {
	*layers.Layers

	app      *tview.Application
	rootFlex *flex.Model // inner + help
	inner    tview.Model
	help     *help.Model

	modalRequest       *ui.ModalMsg
	modalDialog        *modal.Model
	modalPreviousFocus tview.Model
	cfg                *config.Config
}

func NewModel(cfg *config.Config, app *tview.Application) *Model {
	m := &Model{
		Layers:   layers.New(),
		app:      app,
		rootFlex: flex.NewModel(),
		help:     help.NewModel(),

		cfg: cfg,
	}
	m.SetBackgroundLayerStyle(cfg.Theme.Dialog.BackgroundStyle.Style)

	m.rootFlex.SetDirection(flex.DirectionRow)

	styles := help.DefaultStyles()
	styles.ShortKey = cfg.Theme.Help.ShortKeyStyle.Style
	styles.ShortDesc = cfg.Theme.Help.ShortDescStyle.Style
	styles.FullKey = cfg.Theme.Help.FullKeyStyle.Style
	styles.FullDesc = cfg.Theme.Help.FullDescStyle.Style
	m.help.SetStyles(styles)

	m.help.SetKeyMap(m)
	m.help.SetCompactModifiers(cfg.Help.CompactModifiers)
	m.help.SetShortSeparator(cfg.Help.Separator)
	m.help.SetBorderPadding(0, 0, cfg.Help.Padding[0], cfg.Help.Padding[1])
	m.buildLayout()
	return m
}

func (m *Model) showLogin() tview.Cmd {
	m.inner = login.NewModel(m.cfg)
	m.buildLayout()
	return tview.Batch(m.inner.Update(tview.InitMsg{}), tview.SetFocus(m))
}

func (m *Model) showChat(token string) tview.Cmd {
	m.inner = chat.NewModel(m.app, m.cfg, token)
	m.buildLayout()
	return tview.Batch(m.inner.Update(tview.InitMsg{}), tview.SetFocus(m))
}

func (m *Model) buildLayout() {
	m.Clear()
	m.modalRequest = nil
	m.modalDialog = nil
	m.modalPreviousFocus = nil
	m.rootFlex.Clear()
	if m.inner != nil {
		m.rootFlex.AddItem(m.inner, 0, 1, true)
	}
	m.rootFlex.AddItem(m.help, 1, 0, false)
	m.AddLayer(m.rootFlex, layers.WithName(contentLayerName), layers.WithResize(true), layers.WithVisible(true))
	m.updateHelpHeight()
}

var _ tview.Model = (*Model)(nil)

func (m *Model) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case tview.InitMsg:
		var cmd tview.Cmd
		if token := os.Getenv(tokenEnvVarKey); token != "" {
			cmd = tokenCmd(token)
		} else {
			cmd = getToken()
		}
		return tview.Batch(
			tview.SetTitle(consts.Name),
			initClipboard(),
			cmd,
		)

	case loginMsg:
		return m.showLogin()
	case tokenMsg:
		return m.showChat(string(msg))

	case token.TokenMsg:
		return tview.Batch(m.showChat(string(msg)), setToken(string(msg)))
	case qr.TokenMsg:
		return tview.Batch(m.showChat(string(msg)), setToken(string(msg)))

	case chat.LogoutMsg:
		return tview.Batch(
			m.showLogin(),
			deleteToken(),
		)
	case ui.ModalMsg:
		return m.showModal(msg)
	case modal.DoneMsg:
		return m.finishModal(msg)

	case tview.KeyMsg:
		if m.modalRequest != nil {
			if !m.modalDialog.HasFocus() {
				return tview.Sequence(tview.SetFocus(m.modalDialog), func() tview.Msg { return msg })
			}
			return m.Layers.Update(msg)
		}
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.ToggleHelp.Keybind):
			m.help.SetShowAll(!m.help.ShowAll())
			m.updateHelpHeight()
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.Suspend.Keybind):
			m.suspend()
			return nil
		case keybind.Matches(msg, m.cfg.Keybinds.Quit.Keybind):
			var innerCmd tview.Cmd
			if m.inner != nil {
				innerCmd = m.inner.Update(chat.QuitMsg{})
			}
			return tview.Sequence(innerCmd, tview.Quit())
		}
	case tview.MouseMsg, tview.PasteMsg, tview.FormSubmitMsg, tview.FormCancelMsg, tview.ButtonExitMsg:
		if m.modalRequest != nil {
			return m.Layers.Update(msg)
		}
	}

	if m.inner != nil {
		return m.inner.Update(msg)
	}
	return nil
}

func (m *Model) showModal(request ui.ModalMsg) tview.Cmd {
	if m.modalRequest != nil {
		return nil
	}

	labels := make([]string, len(request.Buttons))
	for i, button := range request.Buttons {
		labels[i] = button.Label
	}
	dialog := modal.NewModel().SetText(request.Text).AddButtons(labels)
	style := m.cfg.Theme.Dialog.Style.Style
	if bg := style.GetBackground(); bg != tcell.ColorDefault {
		dialog.SetBackgroundColor(bg)
	}
	if fg := style.GetForeground(); fg != tcell.ColorDefault {
		dialog.SetTextColor(fg)
	}
	dialog.SetButtonStyle(style).SetButtonActivatedStyle(style.Reverse(true))

	m.modalRequest = &request
	m.modalDialog = dialog
	m.modalPreviousFocus = m.app.Focused()
	m.AddLayer(dialog,
		layers.WithName(modalLayerName),
		layers.WithResize(true),
		layers.WithVisible(true),
		layers.WithOverlay(),
	)
	return nil
}

func (m *Model) finishModal(done modal.DoneMsg) tview.Cmd {
	request := m.modalRequest
	if request == nil {
		return nil
	}

	var result tview.Msg
	if done.ButtonIndex >= 0 && done.ButtonIndex < len(request.Buttons) {
		button := request.Buttons[done.ButtonIndex]
		if button.KeepOpen {
			return func() tview.Msg { return button.Result }
		}
		result = button.Result
	}

	m.RemoveLayer(modalLayerName)
	focus := tview.Cmd(nil)
	if m.modalPreviousFocus != nil {
		focus = tview.SetFocus(m.modalPreviousFocus)
	}
	m.modalRequest = nil
	m.modalDialog = nil
	m.modalPreviousFocus = nil
	if result == nil {
		return focus
	}
	return tview.Sequence(focus, func() tview.Msg { return result })
}

func (m *Model) updateHelpHeight() {
	height := 1
	if m.help.ShowAll() {
		height = max(len(m.help.FullHelpLines(m.FullHelp(), 0)), 1)
	}
	m.rootFlex.ResizeItem(m.help, height, 0)
}
