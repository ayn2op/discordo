package logger

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/consts"
)

// Opens the log file and configures default logger.
func Load() error {
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

	l := slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{AddSource: true}))
	slog.SetDefault(l)
	return nil
}
