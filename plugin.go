package main

import (
	"log"
	"os"
	"path/filepath"
	"plugin"
)

type Plugin struct {
	*plugin.Plugin
}

func newPlugin(path string) (*Plugin, error) {
	p, err := plugin.Open(path)
	if err != nil {
		return nil, err
	}

	return &Plugin{Plugin: p}, nil
}

func (p *Plugin) Name() string {
	s, _ := p.Lookup("Name")
	return *(s).(*string)
}

func loadPlugins() error {
	path, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	path = filepath.Join(path, name, "plugins")
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		if filepath.Ext(entry.Name()) == ".so" {
			p, err := newPlugin(filepath.Join(path, entry.Name()))
			if err != nil {
				return err
			}

			plugins = append(plugins, p)
		}
	}

	return nil
}
