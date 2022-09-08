package config

import (
	_ "embed"
	"io"
	"os"
	"path/filepath"

	lua "github.com/yuin/gopher-lua"
)

const Name = "discordo"

//go:embed config.lua
var LuaConfig []byte

// Config initializes a new Lua state, loads a configuration file, and defines essential micellaneous fields.
type Config struct {
	// Path is the path of the configuration file. Its value is the configuration directory until Load() is called.
	Path  string
	State *lua.LState
}

func New(path string) *Config {
	return &Config{
		Path:  path,
		State: lua.NewState(),
	}
}

func (c *Config) Load() error {
	// Create directories that do not exist and are mentioned in the path recursively.
	err := os.MkdirAll(c.Path, os.ModePerm)
	if err != nil {
		return err
	}

	c.Path = filepath.Join(c.Path, "config.lua")
	// Open the existing configuration file with read-only flag.
	f, err := os.Open(c.Path)
	// If the configuration file does not exist, create a new configuration file with the read-write flag.
	if os.IsNotExist(err) {
		f, err = os.Create(c.Path)
		if err != nil {
			return err
		}
		defer f.Close()

		_, err = f.Write(LuaConfig)
		if err != nil {
			return err
		}

		return f.Sync()
	}

	if err != nil {
		return err
	}
	defer f.Close()

	b, err := io.ReadAll(f)
	if err != nil {
		return err
	}

	LuaConfig = b
	return nil
}

func (c *Config) KeyLua(s *lua.LState) int {
	keyTable := s.NewTable()
	keyTable.RawSetString("name", s.Get(1))
	keyTable.RawSetString("description", s.Get(2))
	keyTable.RawSetString("action", s.Get(3))

	s.Push(keyTable) // Push the result
	return 1         // Number of results
}
