package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/consts"
)

const fileName = "logs.txt"

func DefaultPath() string {
	path, err := os.UserCacheDir()
	if err != nil {
		slog.Info(
			"user cache directory path cannot be determined; falling back to the current directory path",
		)
		path = "."
	}

	return filepath.Join(path, consts.Name, fileName)
}

// Load opens the log file and configures default logger.
func Load(path string, level slog.Level) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	opts := &slog.HandlerOptions{AddSource: true, Level: level}
	handler := slog.NewTextHandler(file, opts)
	slog.SetDefault(slog.New(handler))
	return nil
}
