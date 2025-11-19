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
	return filepath.Join(consts.CacheDir(), fileName)
}

func StringToLevel(s string) slog.Level {
	switch s {
	case "debug":
		return slog.LevelDebug
	case "warn":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}

// Load opens the log file and configures default logger.
func Load(path string, level slog.Level) error {
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(file, opts)
	slog.SetDefault(slog.New(handler))
	return nil
}
