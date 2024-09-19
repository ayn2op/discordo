package cmd

import (
	"log/slog"

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
	return app
}

func (app *Application) Show(token string) error {
	if token == "" {
		loginForm := NewLoginForm(func(token string, err error) {
			if err != nil {
				slog.Error("failed to login", "err", err)
				return
			}

			if err := app.Show(token); err != nil {
				slog.Error("failed to show app", "err", err)
			}
		})
		app.SetRoot(loginForm, true)
	} else {
		if err := openState(token); err != nil {
			return err
		}

		app.SetRoot(mainFlex, true)
	}

	return nil
}

func (app *Application) Run(token string) error {
	if err := app.Show(token); err != nil {
		return err
	}

	return app.Application.Run()
}
