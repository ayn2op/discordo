package logger

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/constants"
)

// Recursively creates the log directory if it does not exist already and returns the path to the log file.
func initialize() (string, error) {
	path, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}

	path = filepath.Join(path, constants.Name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", err
	}

	return filepath.Join(path, "logs.txt"), nil
}

// Opens the log file and configures standard logger.
func Load() error {
	path, err := initialize()
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	return nil
}
