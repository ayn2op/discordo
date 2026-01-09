package app

import (
	"fmt"
	"log/slog"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/media"
	"github.com/ayn2op/discordo/internal/ui/chat"
	"github.com/ayn2op/discordo/internal/ui/login"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type App struct {
	inner    *tview.Application
	chatView *chat.View
	cfg      *config.Config
}

func New(cfg *config.Config) *App {
	tview.Styles = tview.Theme{}
	app := &App{
		inner: tview.NewApplication(),
		cfg:   cfg,
	}

	if err := clipboard.Init(); err != nil {
		slog.Error("failed to init clipboard", "err", err)
	}

	app.inner.SetInputCapture(app.onInputCapture)
	return app
}

func (a *App) Run(token string) error {
	proto, err := media.ParseProtocol(a.cfg.ImagePreviews.Type)
	if err != nil {
		slog.Warn("Invalid image protocol config, falling back to auto-detection", "configured", a.cfg.ImagePreviews.Type, "err", err)
	} else if proto == media.ProtoAuto {
		slog.Debug("No protocol configured, using auto-detection")
	} else {
		media.SetProtocol(proto)
	}

	proto = media.DetectProtocol()
	slog.Info("Detected terminal image protocol", "protocol", proto.String())

	screen, err := tcell.NewScreen()
	if err != nil {
		return fmt.Errorf("failed to create screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return fmt.Errorf("failed to init screen: %w", err)
	}

	if a.cfg.Mouse {
		screen.EnableMouse()
	}

	screen.SetTitle(consts.Name)
	screen.EnablePaste()
	screen.EnableFocus()
	a.inner.SetScreen(screen)

	if token == "" {
		loginForm := login.NewForm(a.inner, a.cfg, func(token string) {
			if err := a.showChatView(token); err != nil {
				slog.Error("failed to show chat view", "err", err)
			}
		})
		a.inner.SetRoot(loginForm)
	} else {
		if err := a.showChatView(token); err != nil {
			return err
		}
	}

	return a.inner.Run()
}

func (a *App) showChatView(token string) error {
	a.chatView = chat.NewView(a.inner, a.cfg, a.quit)
	if err := a.chatView.OpenState(token); err != nil {
		return err
	}
	a.inner.SetRoot(a.chatView)
	return nil
}

func (a *App) quit() {
	if a.chatView != nil {
		if err := a.chatView.CloseState(); err != nil {
			slog.Error("failed to close the session", "err", err)
		}
	}

	a.inner.Stop()
}

func (a *App) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case a.cfg.Keys.Quit:
		a.quit()
		return nil
	case "Ctrl+C":
		// https://github.com/ayn2op/tview/blob/a64fc48d7654432f71922c8b908280cdb525805c/application.go#L153
		return tcell.NewEventKey(tcell.KeyCtrlC, "", tcell.ModNone)
	}

	return event
}
