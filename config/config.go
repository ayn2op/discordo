package config

import (
	_ "embed"
	"os"

	lua "github.com/yuin/gopher-lua"
)

//go:embed config.lua
var cfg []byte

type Config struct {
	State *lua.LState
}

func New() *Config {
	return &Config{
		State: lua.NewState(),
	}
}

func (c *Config) Load(path string) error {
	// Create a new configuration file if it does not exist already; otherwise, open the existing file with read-write flag.
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	// If the configuration file is empty, that is, its size is zero, write the default configuration to the file.
	if fi.Size() == 0 {
		f.Write(cfg)
		f.Sync()
	}

	return c.State.DoFile(path)
}

func (c *Config) String(v lua.LValue) string {
	return string(v.(lua.LString))
}

func (c *Config) Bool(v lua.LValue) bool {
	return bool(v.(lua.LBool))
}

func (c *Config) Number(v lua.LValue) float64 {
	return float64(v.(lua.LNumber))
}
