package root

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/gdamore/tcell/v3"
)

type View struct {
	*tview.Box
	app   *tview.Application
	inner tview.Primitive
	chat  *chat.View

	cfg *config.Config
}

func NewView(cfg *config.Config) *View {
	tview.Styles = tview.Theme{}
	v := &View{
		Box: tview.NewBox(),
		app: tview.NewApplication(),
		cfg: cfg,
	}

	if err := clipboard.Init(); err != nil {
		slog.Error("failed to init clipboard", "err", err)
	}

	return v
}

func (v *View) Run() error {
	token := os.Getenv("DISCORDO_TOKEN")
	if token == "" {
		t, err := keyring.GetToken()
		if err != nil {
			slog.Info("failed to retrieve token from keyring", "err", err)
		}
		token = t
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return fmt.Errorf("failed to init screen: %w", err)
	}

	if v.cfg.Mouse {
		screen.EnableMouse()
	}

	screen.SetTitle(consts.Name)
	screen.EnablePaste()
	screen.EnableFocus()
	v.app.SetScreen(screen)
	v.app.SetRoot(v)

	if token == "" {
		v.showLoginView()
	} else {
		if err := v.showChatView(token); err != nil {
			return err
		}
	}

	v.app.SetFocus(v)
	err = v.app.Run()
	v.closeChatViewState()
	return err
}

func (v *View) showLoginView() {
	loginForm := login.NewForm(v.app, v.cfg, func(token string) {
		if err := v.showChatView(token); err != nil {
			slog.Error("failed to show chat view", "err", err)
		}
	})
	v.inner = loginForm
	v.MarkDirty()
	v.app.SetFocus(v)
}

func (v *View) showChatView(token string) error {
	v.chat = chat.NewView(v.app, v.cfg, v.showLoginView)
	if err := v.chat.OpenState(token); err != nil {
		return err
	}
	v.inner = v.chat
	v.MarkDirty()
	v.app.SetFocus(v)
	return nil
}

func (v *View) quit() {
	v.closeChatViewState()
	v.app.Stop()
}

func (v *View) closeChatViewState() {
	if v.chat != nil {
		if err := v.chat.CloseState(); err != nil {
			slog.Error("failed to close the session", "err", err)
		}
		v.chat = nil
	}
}

func (v *View) Draw(screen tcell.Screen) {
	if v.inner == nil {
		return
	}
	x, y, width, height := v.GetRect()
	v.inner.SetRect(x, y, width, height)
	v.inner.Draw(screen)
}

func (v *View) InputHandler(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	switch {
	case keybind.Matches(event, v.cfg.Keybinds.Suspend.Keybind):
		v.suspend()
		return
	case keybind.Matches(event, v.cfg.Keybinds.Quit.Keybind):
		v.quit()
		return
	}

	if v.inner != nil {
		v.inner.InputHandler(event, setFocus)
	}
}

func (v *View) Focus(delegate func(p tview.Primitive)) {
	if v.inner != nil {
		delegate(v.inner)
		return
	}
	v.Box.Focus(delegate)
}

func (v *View) HasFocus() bool {
	if v.inner != nil && v.inner.HasFocus() {
		return true
	}
	return v.Box.HasFocus()
}

func (v *View) Blur() {
	if v.inner != nil {
		v.inner.Blur()
	}
	v.Box.Blur()
}

func (v *View) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return v.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		if v.inner == nil {
			return false, nil
		}
		handler := v.inner.MouseHandler()
		if handler == nil {
			return false, nil
		}
		return handler(action, event, setFocus)
	})
}

func (v *View) PasteHandler() func(text string, setFocus func(p tview.Primitive)) {
	return v.WrapPasteHandler(func(text string, setFocus func(p tview.Primitive)) {
		if v.inner == nil {
			return
		}
		handler := v.inner.PasteHandler()
		if handler == nil {
			return
		}
		handler(text, setFocus)
	})
}

func (v *View) IsDirty() bool {
	if v.Box.IsDirty() {
		return true
	}
	if v.inner == nil {
		return false
	}
	return v.inner.IsDirty()
}

func (v *View) MarkClean() {
	v.Box.MarkClean()
	if v.inner != nil {
		v.inner.MarkClean()
	}
}
