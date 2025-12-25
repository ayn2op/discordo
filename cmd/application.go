package cmd

import (
	"fmt"
	"log/slog"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type application struct {
	*tview.Application
	chatView *chatView
	cfg      *config.Config
}

func newApplication(cfg *config.Config) *application {
	tview.Styles = tview.Theme{}
	app := &application{
		Application: tview.NewApplication(),
		cfg:         cfg,
	}

	if err := clipboard.Init(); err != nil {
		slog.Error("failed to init clipboard", "err", err)
	}

	app.SetInputCapture(app.onInputCapture)
	return app
}

func (a *application) run(token string) error {
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
	a.SetScreen(screen)

	if token == "" {
		loginForm := login.NewForm(a.Application, a.cfg, func(token string) {
			if err := a.run(token); err != nil {
				slog.Error("failed to run application", "err", err)
			}
		})
		a.SetRoot(loginForm)
	} else {
		a.chatView = newChatView(a.Application, a.cfg)
		if err := openState(token); err != nil {
			return err
		}
		a.SetRoot(a.chatView)
	}

	return a.Run()
}

func (a *application) quit() {
	if discordState != nil {
		if err := discordState.Close(); err != nil {
			slog.Error("failed to close the session", "err", err)
		}
	}

	a.Stop()
}

func (a *application) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
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
