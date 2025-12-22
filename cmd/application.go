package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

type application struct {
	*tview.Application
	chatView  *chatView
	statusBar *statusBar
	cfg       *config.Config
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

	app.
		EnableMouse(cfg.Mouse).
		SetInputCapture(app.onInputCapture).
		SetBeforeDrawFunc(app.onBeforeDraw).
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
		a.SetRoot(loginForm, true)
	} else {
		a.chatView = newChatView(a.Application, a.cfg)
		if err := openState(token); err != nil {
			return err
		}

		rootFlex := tview.NewFlex().
			SetDirection(tview.FlexRow).
			AddItem(a.chatView, 0, 1, true)
		if a.cfg.ShowStatusBar {
			a.statusBar = newStatusBar(a.cfg)
			rootFlex.AddItem(a.statusBar, 1, 1, false)
		}

		a.SetRoot(rootFlex, true)
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

func (a *application) onBeforeDraw(screen tcell.Screen) bool {
	if a.statusBar != nil {
		a.statusBar.Update(a)
	}
	return false
}
