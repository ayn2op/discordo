package root

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/gdamore/tcell/v3"
)

const tokenEnvVarKey = "DISCORDO_TOKEN"

type View struct {
	app   *tview.Application
	inner tview.Primitive
	chat  *chat.View

	cfg *config.Config
}

func NewView(cfg *config.Config, app *tview.Application) *View {
	return &View{
		app: app,
		cfg: cfg,
	}
}

func (v *View) showLoginView() {
	loginForm := login.NewForm(v.app, v.cfg, func(token string) {
		if err := v.showChatView(token); err != nil {
			slog.Error("failed to show chat view", "err", err)
		}
	})
	v.chat = nil
	v.inner = loginForm
}

func (v *View) showChatView(token string) error {
	v.chat = chat.NewView(v.app, v.cfg)
	if err := v.chat.OpenState(token); err != nil {
		return err
	}
	v.inner = v.chat
	return nil
}

func (v *View) closeChatViewState() {
	if v.chat != nil {
		if err := v.chat.CloseState(); err != nil {
			slog.Error("failed to close the session", "err", err)
		}
		v.chat = nil
	}
}

var _ tview.Primitive = (*View)(nil)

func (v *View) Draw(screen tcell.Screen) {
	if v.inner != nil {
		v.inner.Draw(screen)
	}
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
			v.showLoginView()
		} else {
			if err := v.showChatView(event.token); err != nil {
				slog.Error("failed to show chat view", "err", err)
				return tview.QuitCommand{}
			}
		}
		return tview.BatchCommand{tview.SetFocusCommand{Target: v.inner}}
	case *chat.LogoutEvent:
		v.showLoginView()
		return tview.BatchCommand{
			tview.EventCommand(deleteToken),
			tview.SetFocusCommand{Target: v.inner},
		}

	case *tview.KeyEvent:
		switch {
		case keybind.Matches(event, v.cfg.Keybinds.Suspend.Keybind):
			v.suspend()
			return nil
		case keybind.Matches(event, v.cfg.Keybinds.Quit.Keybind):
			v.closeChatViewState()
			return tview.QuitCommand{}
		}
	}

	if v.inner != nil {
		return v.inner.HandleEvent(event)
	}
	return nil
}

func (v *View) GetRect() (int, int, int, int) {
	if v.inner != nil {
		return v.inner.GetRect()
	}
	return 0, 0, 0, 0
}

func (v *View) SetRect(x int, y int, width int, height int) {
	if v.inner != nil {
		v.inner.SetRect(x, y, width, height)
	}
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
