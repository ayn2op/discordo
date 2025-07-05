// Package cmd defines the command-line interface commands
package cmd

import (
	"fmt"
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v2"
	"github.com/spf13/cobra"
	"github.com/zalando/go-keyring"
)

var (
	discordState *ningen.State
	app          *application

	rootCmd = &cobra.Command{
		Use: consts.Name,
		RunE: func(cmd *cobra.Command, _ []string) error {
			flags := cmd.Flags()

			logLevel, _ := flags.GetString("log-level")

			var level slog.Level
			switch logLevel {
			case "debug":
				ws.EnableRawEvents = true
				level = slog.LevelDebug
			case "info":
				level = slog.LevelInfo
			case "warn":
				level = slog.LevelWarn
			case "error":
				level = slog.LevelError
			}

			logPath, _ := flags.GetString("log-path")
			if err := logger.Load(logPath, level); err != nil {
				return fmt.Errorf("failed to load logger: %w", err)
			}

			configPath, _ := flags.GetString("config-path")
			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			token, _ := flags.GetString("token")
			if token == "" {
				token, err = keyring.Get(consts.Name, "token")
				if err != nil {
					slog.Info("failed to retrieve token from keyring", "err", err)
				}
			}

			tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfg.Theme.BackgroundColor)
			app = newApplication(cfg)
			return app.run(token)
		},
	}

	Execute = rootCmd.Execute
)

func init() {
	flags := rootCmd.Flags()
	flags.StringP("token", "t", "", "authentication token")
	flags.String("config-path", config.DefaultPath(), "path of the configuration file")
	flags.String("log-path", logger.DefaultPath(), "path of the log file")
	flags.String("log-level", "info", "log level")
}
