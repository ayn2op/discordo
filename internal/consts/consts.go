package consts

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"
)

const Name = "discordo"

// ConfigDir returns the directory of the files the user and discordo both own, such as the config and the favorites.
// It is not created here, as a user without one still gets the defaults.
var ConfigDir = sync.OnceValue(func() string {
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		userConfigDir = "."
		slog.Info("failed to get user config dir; falling back to the current dir", "err", err, "path", userConfigDir)
	}

	return filepath.Join(userConfigDir, Name)
})

// CacheDir returns the cache directory, creating it on first use.
var CacheDir = sync.OnceValue(func() string {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		userCacheDir = os.TempDir()
		slog.Warn("failed to get user cache dir; falling back to temp dir", "err", err, "path", userCacheDir)
	}

	path := filepath.Join(userCacheDir, Name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		slog.Error("failed to create cache dir", "err", err, "path", path)
	}

	return path
})
