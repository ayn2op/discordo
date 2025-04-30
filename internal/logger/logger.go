package logger

import (
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/consts"
)

const fileName = "logs.txt"

// Opens the log file and configures default logger.
func Load(level slog.Level) error {
	path, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	path = filepath.Join(path, consts.Name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	path = filepath.Join(path, fileName)
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	l := slog.New(slog.NewTextHandler(file, &slog.HandlerOptions{AddSource: true, Level: level}))
	slog.SetDefault(l)
	return nil
}
