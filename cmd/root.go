// Package cmd defines the command-line interface commands
package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
	"github.com/zalando/go-keyring"
)

var (
	discordState *ningen.State
	app          *application
)

func Run() error {
	tokenEnvVar := os.Getenv("DISCORDO_TOKEN")
	tokenFlag := flag.String("token", tokenEnvVar, "authentication token")

	configPath := flag.String("config-path", config.DefaultPath(), "path of the configuration file")
	logPath := flag.String("log-path", logger.DefaultPath(), "path of the log file")
	logLevel := flag.String("log-level", "info", "log level")
	flag.Parse()

	var level slog.Level
	switch *logLevel {
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

	if err := logger.Load(*logPath, level); err != nil {
		return fmt.Errorf("failed to load logger: %w", err)
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	token := *tokenFlag
	if token == "" {
		token, err = keyring.Get(consts.Name, "token")
		if err != nil {
			slog.Info("failed to retrieve token from keyring", "err", err)
		}
	}

	tview.Styles = tview.Theme{}
	app = newApplication(cfg)
	return app.run(token)
}
