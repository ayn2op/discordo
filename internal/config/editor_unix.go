//go:build unix

package config

import (
	"os/exec"
	"fmt"
)

func (cfg *Config) createEditorCommand(path string) *exec.Cmd {
	// return exec.Command("sh", "-c", cfg.Editor+" \"$@\"", cfg.Editor, path)
	return exec.Command("sh", "-c", fmt.Sprintf("%s '%s'", cfg.Editor, path))
}
