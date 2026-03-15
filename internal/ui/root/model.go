package root

import (
	"os"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	"github.com/ayn2op/discordo/internal/ui/login/qr"
	"github.com/ayn2op/discordo/internal/ui/login/token"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/flex"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/gdamore/tcell/v3"
)

const tokenEnvVarKey = "DISCORDO_TOKEN"

type Model struct {
	app      *tview.Application
	rootFlex *flex.Model // inner + help
	inner    tview.Primitive
	help     *help.Help

	cfg *config.Config
}

func NewModel(cfg *config.Config, app *tview.Application) *Model {
	m := &Model{
		app:      app,
		rootFlex: flex.NewModel(),
		help:     help.New(),

		cfg: cfg,
	}

	m.rootFlex.SetDirection(flex.DirectionRow)

	styles := help.DefaultStyles()
	styles.ShortKeyStyle = cfg.Theme.Help.ShortKeyStyle.Style
	styles.ShortDescStyle = cfg.Theme.Help.ShortDescStyle.Style
	styles.FullKeyStyle = cfg.Theme.Help.FullKeyStyle.Style
	styles.FullDescStyle = cfg.Theme.Help.FullDescStyle.Style
	m.help.SetStyles(styles)

	m.help.SetKeyMap(m)
	m.help.SetCompactModifiers(cfg.Help.CompactModifiers)
	m.help.SetShortSeparator(cfg.Help.Separator)
	m.help.SetBorderPadding(0, 0, cfg.Help.Padding[0], cfg.Help.Padding[1])
	m.buildLayout()
	return m
}

func (m *Model) showLogin() tview.Command {
	m.inner = login.NewModel(m.cfg)
	m.buildLayout()
	return tview.Batch(m.inner.HandleEvent(&tview.InitEvent{}), tview.SetFocus(m))
}

func (m *Model) showChat(token string) tview.Command {
	m.inner = chat.NewView(m.app, m.cfg, token)
	m.buildLayout()
	return tview.Batch(m.inner.HandleEvent(&tview.InitEvent{}), tview.SetFocus(m))
}

func (m *Model) buildLayout() {
	m.rootFlex.Clear()
	if m.inner != nil {
		m.rootFlex.AddItem(m.inner, 0, 1, true)
	}
	m.rootFlex.AddItem(m.help, 1, 0, false)
	m.updateHelpHeight()
}

var _ tview.Primitive = (*Model)(nil)

func (m *Model) Draw(screen tcell.Screen) {
	m.rootFlex.Draw(screen)
}

func (m *Model) HandleEvent(event tcell.Event) tview.Command {
	switch event := event.(type) {
	case *tview.InitEvent:
		var cmd tview.Command
		if token := os.Getenv(tokenEnvVarKey); token != "" {
			cmd = tokenCommand(token)
		} else {
			cmd = getToken()
		}
		return tview.Batch(
			tview.SetTitle(consts.Name),
			initClipboard(),
			cmd,
		)

	case *loginEvent:
		return m.showLogin()
	case *tokenEvent:
		return m.showChat(event.token)

	case *token.TokenEvent:
		return tview.Batch(m.showChat(event.Token), setToken(event.Token))
	case *qr.TokenEvent:
		return tview.Batch(m.showChat(event.Token), setToken(event.Token))

	case *chat.LogoutEvent:
		return tview.Batch(
			m.showLogin(),
			deleteToken(),
		)

	case *tview.KeyEvent:
		switch {
		case keybind.Matches(event, m.cfg.Keybinds.ToggleHelp.Keybind):
			m.help.SetShowAll(!m.help.ShowAll())
			m.updateHelpHeight()
			return nil
		case keybind.Matches(event, m.cfg.Keybinds.Suspend.Keybind):
			m.suspend()
			return nil
		case keybind.Matches(event, m.cfg.Keybinds.Quit.Keybind):
			var innerCmd tview.Command
			if m.inner != nil {
				innerCmd = m.inner.HandleEvent(&chat.QuitEvent{})
			}
			return tview.Batch(innerCmd, tview.Quit())
		}
	}

	if m.inner != nil {
		return m.inner.HandleEvent(event)
	}
	return nil
}

func (m *Model) updateHelpHeight() {
	height := 1
	if m.help.ShowAll() {
		height = max(len(m.help.FullHelpLines(m.FullHelp(), 0)), 1)
	}
	m.rootFlex.ResizeItem(m.help, height, 0)
}

func (m *Model) GetRect() (int, int, int, int) {
	return m.rootFlex.GetRect()
}

func (m *Model) SetRect(x int, y int, width int, height int) {
	m.rootFlex.SetRect(x, y, width, height)
}

func (m *Model) Focus(delegate func(p tview.Primitive)) {
	if m.inner != nil {
		delegate(m.inner)
	}
}

func (m *Model) HasFocus() bool {
	if m.inner != nil {
		return m.inner.HasFocus()
	}
	return true
}

func (m *Model) Blur() {
	if m.inner != nil {
		m.inner.Blur()
	}
}
