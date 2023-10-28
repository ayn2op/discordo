package config

import (
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/constants"
)

func DefaultLogPath() string {
	path, _ := os.UserCacheDir()
	return filepath.Join(path, constants.Name, "logs.txt")
}
