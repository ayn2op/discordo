//go:build unix

package config

import (
	"log/slog"
	"os/exec"
)

func (cfg *Config) OpenFile(path string) *exec.Cmd {
	if cfg.Editor == "" {
		slog.Warn("Attempt to open file with editor, but no editor is set")
		return nil
	}

	return exec.Command("sh", "-c", cfg.Editor+" \"$@\"", cfg.Editor, path)
}
