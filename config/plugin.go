package config

import (
	"os"
	"path/filepath"

	"github.com/d5/tengo/v2"
	"github.com/pkg/errors"
)

var Plugins []*Plugin

// LoadPlugins reads the plugins directory and loads all of the plugins inside it. It also creates the plugins directory if it does not exist already.
func LoadPlugins() error {
	path, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	path = filepath.Join(path, Name, "plugins")
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	var errs error
	for _, entry := range entries {
		input, err := os.ReadFile(filepath.Join(path, entry.Name()))
		if err != nil {
			errs = errors.Wrap(errs, err.Error())
			continue
		}

		p, err := NewPlugin(input)
		if err != nil {
			errs = errors.Wrap(errs, err.Error())
		}

		Plugins = append(Plugins, p)
	}

	return errs
}

type Plugin struct {
	script *tengo.Script
}

func NewPlugin(input []byte) (*Plugin, error) {
	p := &Plugin{
		script: tengo.NewScript(input),
	}

	_, err := p.script.Run()
	if err != nil {
		return nil, err
	}

	return p, nil
}
