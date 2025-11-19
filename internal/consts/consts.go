package consts

import (
	"log/slog"
	"os"
	"path/filepath"
)

const (
	Name        = "discordo"
	Description = "A lightweight, secure, and feature-rich Discord terminal (TUI) client."
)

var cacheDir string

func CacheDir() string {
	return cacheDir
}

func init() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		userCacheDir = os.TempDir()
		slog.Warn("failed to get user cache dir; falling back to temp dir", "err", err, "path", userCacheDir)
	}

	cacheDir = filepath.Join(userCacheDir, Name)
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		slog.Error("failed to create cache dir", "err", err, "path", cacheDir)
	}
}
