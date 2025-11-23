package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
	"golang.design/x/clipboard"
)

const (
	flexPageName            = "flex"
	mentionsListPageName    = "mentionsList"
	attachmentsListPageName = "attachmentsList"
	confirmModalPageName    = "confirmModal"
)

type application struct {
	*tview.Application
	chatView *chatView
	cfg      *config.Config
}

func newApplication(cfg *config.Config) *application {
	app := &application{
		Application: tview.NewApplication(),
		cfg:         cfg,
	}

	if err := clipboard.Init(); err != nil {
		slog.Error("failed to init clipboard", "err", err)
	}

	app.
		EnableMouse(cfg.Mouse).
		SetInputCapture(app.onInputCapture).
		EnablePaste(true)
	return app
}

func (a *application) run(token string) error {
	if token == "" {
		loginForm := login.NewForm(a.Application, a.cfg, func(token string) {
			if err := a.run(token); err != nil {
				slog.Error("failed to run application", "err", err)
			}
		})

		return a.SetRoot(loginForm, true).Run()
	}

	a.chatView = newChatView(a.Application, a.cfg)
	if err := openState(token); err != nil {
		return err
	}

	return a.SetRoot(a.chatView, true).Run()
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
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	}

	return event
}
