package cmd

import (
	"log/slog"

	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Application struct {
	*tview.Application
}

func newApplication() *Application {
	app := &Application{
		Application: tview.NewApplication(),
	}

	app.EnableMouse(cfg.Mouse)
	app.SetInputCapture(app.onInputCapture)
	return app
}

func (app *Application) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.Quit:
		app.Stop()
	case "Ctrl+C":
		// https://github.com/rivo/tview/blob/a64fc48d7654432f71922c8b908280cdb525805c/application.go#L153
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	}

	return event
}

func (app *Application) show(token string) error {
	if token == "" {
		loginForm := newLoginForm(func(token string, err error) {
			if err != nil {
				slog.Error("failed to login", "err", err)
				return
			}

			if err := app.show(token); err != nil {
				slog.Error("failed to show app", "err", err)
			}
		})
		app.SetRoot(loginForm, true)
	} else {
		// mainFlex must be initialized before opening a new state.
		mainFlex = newMainFlex()
		if err := openState(token); err != nil {
			return err
		}

		app.SetRoot(mainFlex, true)
	}

	return nil
}

func (app *Application) Run(token string) error {
	if err := app.show(token); err != nil {
		return err
	}

	return app.Application.Run()
}
