package logger

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/consts"
)

// Load opens the log file and configures default logger.
func Load(level slog.Level) error {
	path, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	path = filepath.Join(path, consts.Name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	path = filepath.Join(path, "logs.txt")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	opts := &slog.HandlerOptions{AddSource: true, Level: level}
	handler := slog.NewTextHandler(file, opts)
	slog.SetDefault(slog.New(handler))
	return nil
}
