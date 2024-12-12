package logger

import (
	"os"
	"path/filepath"
	"time"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/charmbracelet/log"
)

// Opens the log file and configures the default logger.
func Load() error {
	log.SetReportTimestamp(true)
	log.SetReportCaller(true)
	log.SetTimeFormat(time.Kitchen)

	path, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	path = filepath.Join(path, config.Name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return err
	}

	path = filepath.Join(path, "logs.txt")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	log.SetOutput(file)
	return nil
}
