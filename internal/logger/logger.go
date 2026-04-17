package logger

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/ayn2op/discordo/internal/consts"
)

const fileName = "logs.txt"

var (
	logFile *os.File
	mu      sync.Mutex
)

func DefaultPath() string {
	return filepath.Join(consts.CacheDir(), fileName)
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

	mu.Lock()
	logFile = file
	mu.Unlock()

	opts := &slog.HandlerOptions{Level: level}
	handler := slog.NewTextHandler(file, opts)
	slog.SetDefault(slog.New(handler))
	return nil
}

// Close closes the log file to prevent resource leaks.
func Close() error {
	mu.Lock()
	defer mu.Unlock()

	if logFile != nil {
		return logFile.Close()
	}
	return nil
}
