package config

import (
	"errors"
	"os/exec"

	"github.com/mattn/go-shellwords"
)

func (cfg *Config) EditorCommand(path string) (*exec.Cmd, error) {
	args, err := shellwords.Parse(cfg.Editor)
	if err != nil {
		return nil, err
	}
	if len(args) == 0 {
		return nil, errors.New("editor command is empty")
	}
	return exec.Command(args[0], append(args[1:], path)...), nil
}
