package config

import (
	"log"
	"os"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

var Plugins []*Plugin

// LoadPlugins reads the plugins directory and loads all of the plugins inside it. It creates the plugins directory if it does not exist already.
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

	for _, entry := range entries {
		if filepath.Ext(entry.Name()) == ".lua" {
			p, err := loadPlugin(filepath.Join(path, entry.Name()))
			if err != nil {
				// TODO: multiple errors
				log.Println(err)
				continue
			}

			Plugins = append(Plugins, p)
		}
	}

	return err
}

// CallPlugins calls a function present within the global scope for all loaded plugins.
func CallPlugins(name string, args ...any) {
	for _, p := range Plugins {
		vargs := make([]lua.LValue, len(args))
		for i, arg := range args {
			vargs[i] = luar.New(p.state, arg)
		}

		_ = p.state.CallByParam(lua.P{
			Fn:      p.state.GetGlobal(name),
			Protect: true,
		}, vargs...)
	}
}

type Plugin struct {
	state *lua.LState
}

func loadPlugin(path string) (*Plugin, error) {
	p := &Plugin{
		state: lua.NewState(),
	}

	err := p.state.DoFile(path)
	if err != nil {
		return nil, err
	}

	return p, nil
}
