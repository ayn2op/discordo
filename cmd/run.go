package cmd

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/logger"
)

var (
	discordState *State
	mainFlex     *Layout
)

func Run(token string) error {
	if err := logger.Load(); err != nil {
		return err
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	mainFlex = newLayout(cfg)
	return mainFlex.run(token)
}
