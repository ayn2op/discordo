package config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/gdamore/tcell/v2"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

var Plugins []*Plugin

type Plugin struct {
	state *lua.LState
}

func openPlugin(path string) (*Plugin, error) {
	p := &Plugin{
		state: lua.NewState(),
	}

	err := p.state.DoFile(path)
	if err != nil {
		return nil, err
	}

	return p, nil
}

func (p *Plugin) OnInputCapture(widget string, event *tcell.EventKey) {
	_ = p.state.CallByParam(lua.P{
		Fn:      p.state.GetGlobal("onInputCapture"),
		Protect: true,
	}, luar.New(p.state, widget), luar.New(p.state, event))
}

// LoadPlugins reads the plugins directory and opens the plugins inside it. It creates the plugins directory if it does not exist already.
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
			p, err := openPlugin(filepath.Join(path, entry.Name()))
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
