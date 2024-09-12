package cmd

import (
	"log"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/rivo/tview"
)

var (
	discordState *State

	cfg      *config.Config
	app      = tview.NewApplication()
	mainFlex *MainFlex
)

func Run(token string) error {
	if err := logger.Load(); err != nil {
		return err
	}

	var err error
	cfg, err = config.Load()
	if err != nil {
		return err
	}

	// mainFlex must be initialized before opening a new state.
	mainFlex = newMainFlex()
	if token == "" {
		lf := NewLoginForm(func(token string, err error) {
			if err != nil {
				app.Stop()
				log.Fatal(err)
			}

			if err := openState(token); err != nil {
				app.Stop()
				log.Fatal(err)
			}

			app.SetRoot(mainFlex, true)
		})

		app.SetRoot(lf, true)
	} else {
		if err := openState(token); err != nil {
			return err
		}

		app.SetRoot(mainFlex, true)
	}

	app.EnableMouse(cfg.Mouse)
	return app.Run()
}
