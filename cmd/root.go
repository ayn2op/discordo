// Package cmd defines the command-line interface commands
package cmd

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
	"github.com/spf13/cobra"
)

var (
	discordState *ningen.State
	app          *application
)

var (
	token      string
	configPath string
	logPath    string
	logLevel   string

	rootCmd = &cobra.Command{
		Use: consts.Name,
		RunE: func(cmd *cobra.Command, args []string) error {
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

			if err := logger.Load(logPath, level); err != nil {
				return fmt.Errorf("failed to load logger: %w", err)
			}

			cfg, err := config.Load(configPath)
			if err != nil {
				return fmt.Errorf("failed to load config: %w", err)
			}

			if token == "" {
				token = os.Getenv("DISCORDO_TOKEN")
			}

			if token == "" {
				token, err = keyring.GetToken()
				if err != nil {
					slog.Info("failed to retrieve token from keyring", "err", err)
				}
			}

			app = newApplication(cfg)
			return app.run(token)
		},
	}

	Execute = rootCmd.Execute
)

func init() {
	flags := rootCmd.Flags()
	flags.StringVar(&token, "token", "", "authentication token (default: $DISCORDO_TOKEN or keyring)")

	flags.StringVar(&configPath, "config-path", config.DefaultPath(), "path of the configuration file")

	flags.StringVar(&logPath, "log-path", logger.DefaultPath(), "path of the log file")
	flags.StringVar(&logLevel, "log-level", "info", "log level")
}
