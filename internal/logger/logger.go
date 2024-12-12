package logger

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/config"
)

// Recursively creates the log directory if it does not exist already and returns the path to the log file.
func initialize() (string, error) {
	path, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	path = filepath.Join(path, config.Name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", err
	}

	return filepath.Join(path, "logs.txt"), nil
}

// Opens the log file and configures standard logger.
func Load() error {
	path, err := initialize()
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	l := slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{AddSource: true}))
	slog.SetDefault(l)
	return nil
}
