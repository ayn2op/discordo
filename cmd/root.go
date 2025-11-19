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
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
	"github.com/spf13/cobra"
)

var (
	discordState *ningen.State
	app          *application
)

var (
	token string

	configPath string
	logPath    string
	logLevel   string

	Execute = rootCmd.Execute
)

var rootCmd = &cobra.Command{
	Use:   consts.Name,
	Short: consts.Description,
	RunE: func(cmd *cobra.Command, args []string) error {
		level := logger.StringToLevel(logLevel)
		if level == slog.LevelDebug {
			ws.EnableRawEvents = true
		}

		if err := logger.Load(logPath, level); err != nil {
			return fmt.Errorf("failed to load logger: %w", err)
		}

		cfg, err := config.Load(configPath)
		if err != nil {
			return fmt.Errorf("failed to load config: %w", err)
		}

		if token == "" {
			token, err = keyring.GetToken()
			if err != nil {
				slog.Info("failed to retrieve token from keyring", "err", err)
			}
		}

		tview.Styles = tview.Theme{}
		app = newApplication(cfg)
		return app.run(token)
	},
}

func init() {
	flags := rootCmd.Flags()
	flags.StringVarP(&token, "token", "t", os.Getenv("DISCORDO_TOKEN"), "authentication token")

	flags.StringVar(&configPath, "config-path", config.DefaultPath(), "path of configuration file")
	flags.StringVar(&logPath, "log-path", logger.DefaultPath(), "path of log file")
	flags.StringVar(&logLevel, "log-level", "info", "log level")
}
