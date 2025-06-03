package logger

import (
	"log/slog"
	"os"
	"path/filepath"

	"tui/internal/consts"
)

type Format int

const (
	FormatText Format = iota
	FormatJson
)

// Opens the log file and configures default logger.
func Load(format Format, level slog.Level) error {
	path, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	path = filepath.Join(path, consts.Name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	opts := &slog.HandlerOptions{AddSource: true, Level: level}

	var h slog.Handler
	switch format {
	case FormatText:
		path := filepath.Join(path, "logs.txt")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return err
		}

		h = slog.NewTextHandler(file, opts)
	case FormatJson:
		path := filepath.Join(path, "logs.jsonl")
		file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
		if err != nil {
			return err
		}

		h = slog.NewJSONHandler(file, opts)
	}

	slog.SetDefault(slog.New(h))
	return nil
}
