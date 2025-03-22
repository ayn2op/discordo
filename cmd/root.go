package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var (
	discordState *State
	app          *App
)

var rootCmd = &cobra.Command{
	RunE: func(cmd *cobra.Command, args []string) error {
		if err := logger.Load(); err != nil {
			return err
		}

		token, _ := cmd.Flags().GetString("token")
		if token == "" {
			var err error
			token, err = keyring.Get(consts.Name, "token")
			if err != nil {
				slog.Info("failed to retrieve token from keyring", "err", err)
			}
		}

		cfg, err := config.Load()
		if err != nil {
			return err
		}

		app = newApp(cfg)
		return app.run(token)
	},
}

var Execute = rootCmd.Execute

func init() {
	rootCmd.Flags().StringP("token", "t", "", "the authentication token")
}
