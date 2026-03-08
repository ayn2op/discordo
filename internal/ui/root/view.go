package root

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/gdamore/tcell/v3"
)

const tokenEnvVarKey = "DISCORDO_TOKEN"

type View struct {
	app      *tview.Application
	rootFlex *tview.Flex // inner + help
	inner    tview.Primitive
	help     *help.Help

	cfg *config.Config
}

func NewView(cfg *config.Config, app *tview.Application) *View {
	v := &View{
		app:      app,
		rootFlex: tview.NewFlex(),
		help:     help.New(),

		cfg: cfg,
	}

	v.rootFlex.SetDirection(tview.FlexRow)

	styles := help.DefaultStyles()
	styles.ShortKeyStyle = cfg.Theme.Help.ShortKeyStyle.Style
	styles.ShortDescStyle = cfg.Theme.Help.ShortDescStyle.Style
	styles.FullKeyStyle = cfg.Theme.Help.FullKeyStyle.Style
	styles.FullDescStyle = cfg.Theme.Help.FullDescStyle.Style
	v.help.SetStyles(styles)

	v.help.SetKeyMap(v)
	v.help.SetCompactModifiers(cfg.Help.CompactModifiers)
	v.help.SetShortSeparator(cfg.Help.Separator)
	v.help.SetBorderPadding(0, 0, cfg.Help.Padding[0], cfg.Help.Padding[1])
	v.buildLayout()
	return v
}

func (v *View) showLoginView() tview.Command {
	v.inner = login.NewForm(v.app, v.cfg)
	v.buildLayout()
	return v.inner.HandleEvent(tview.NewInitEvent())
}

func (v *View) showChatView(token string) tview.Command {
	v.inner = chat.NewView(v.app, v.cfg, token)
	v.buildLayout()
	return v.inner.HandleEvent(tview.NewInitEvent())
}

func (v *View) buildLayout() {
	v.rootFlex.Clear()

	content := v.inner
	if content == nil {
		content = tview.NewBox()
	}
	v.rootFlex.AddItem(content, 0, 1, true)
	v.rootFlex.AddItem(v.help, 1, 0, false)
	v.updateHelpHeight()
}

var _ tview.Primitive = (*View)(nil)

func (v *View) Draw(screen tcell.Screen) {
	v.rootFlex.Draw(screen)
}

func (v *View) HandleEvent(event tcell.Event) tview.Command {
	switch event := event.(type) {
	case *tview.InitEvent:
		return tview.BatchCommand{
			tview.SetTitleCommand(consts.Name),
			tview.EventCommand(initClipboard),
			tview.EventCommand(getToken),
		}
	case *tokenEvent:
		if event.token == "" {
			return tview.BatchCommand{v.showLoginView(), tview.SetFocusCommand{Target: v.inner}}
		} else {
			return tview.BatchCommand{v.showChatView(event.token), tview.SetFocusCommand{Target: v.inner}}
		}
	case *login.TokenEvent:
		return tview.BatchCommand{v.showChatView(event.Token), tview.SetFocusCommand{Target: v.inner}}
	case *chat.LogoutEvent:
		v.showLoginView()
		return tview.BatchCommand{
			tview.EventCommand(deleteToken),
			tview.SetFocusCommand{Target: v.inner},
		}

	case *tview.KeyEvent:
		switch {
		case keybind.Matches(event, v.cfg.Keybinds.ToggleHelp.Keybind):
			v.help.SetShowAll(!v.help.ShowAll())
			v.updateHelpHeight()
			return tview.RedrawCommand{}
		case keybind.Matches(event, v.cfg.Keybinds.Suspend.Keybind):
			v.suspend()
			return nil
		case keybind.Matches(event, v.cfg.Keybinds.Quit.Keybind):
			var innerCmd tview.Command
			if v.inner != nil {
				innerCmd = v.inner.HandleEvent(chat.NewQuitEvent())
			}
			return tview.BatchCommand{innerCmd, tview.QuitCommand{}}
		}
	}

	if v.inner != nil {
		return v.inner.HandleEvent(event)
	}
	return nil
}

func (v *View) updateHelpHeight() {
	height := 1
	if v.help.ShowAll() {
		height = max(len(v.help.FullHelpLines(v.FullHelp(), 0)), 1)
	}
	v.rootFlex.ResizeItem(v.help, height, 0)
}

func (v *View) GetRect() (int, int, int, int) {
	return v.rootFlex.GetRect()
}

func (v *View) SetRect(x int, y int, width int, height int) {
	v.rootFlex.SetRect(x, y, width, height)
}

func (v *View) Focus(delegate func(p tview.Primitive)) {
	if v.inner != nil {
		delegate(v.inner)
	}
}

func (v *View) HasFocus() bool {
	if v.inner != nil {
		return v.inner.HasFocus()
	}
	return true
}

func (v *View) Blur() {
	if v.inner != nil {
		v.inner.Blur()
	}
}
