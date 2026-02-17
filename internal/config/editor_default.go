//go:build !unix

package config

import (
	"os/exec"
)

func (cfg *Config) CreateEditorCommand(path string) *exec.Cmd {
	return exec.Command(cfg.Editor, path)
}
