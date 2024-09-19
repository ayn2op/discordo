package cmd

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/logger"
)

var (
	discordState *State

	cfg      *config.Config
	app      *Application
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

	// app must be initialized after configuration is loaded
	app = newApplication()
	// mainFlex must be initialized before opening a new state.
	mainFlex = newMainFlex()
	return app.Run(token)
}
