package cmd

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/ayn2op/discordo/internal/app"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/logger"
	"github.com/diamondburned/arikawa/v3/utils/ws"
)

var (
	token      string
	configPath string
	logPath    string
	logLevel   string
)

func Run() error {
	flag.StringVar(&token, "token", "", "authentication token (default: $DISCORDO_TOKEN or keyring)")
	flag.StringVar(&configPath, "config-path", config.DefaultPath(), "path of the configuration file")
	flag.StringVar(&logPath, "log-path", logger.DefaultPath(), "path of the log file")
	flag.StringVar(&logLevel, "log-level", "info", "log level")
	flag.Parse()

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

	return app.New(cfg).Run(token)
}
