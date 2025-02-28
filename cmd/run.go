package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/zalando/go-keyring"
)

var (
	discordState *State
	app          *App
)

func Run(token string) error {
	if err := logger.Load(); err != nil {
		return err
	}

	// If no token was provided, look it up in the keyring.
	if token == "" {
		tok, err := keyring.Get(consts.Name, "token")
		if err != nil {
			slog.Info("failed to get token from keyring", "err", err)
		} else {
			token = tok
		}
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	app = newApp(cfg)
	return app.run(token)
}
