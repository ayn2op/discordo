package cmd

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/logger"
)

var (
	discordState *State

	cfg      *config.Config
	mainFlex *Layout
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

	mainFlex = newLayout()
	return mainFlex.run(token)
}
