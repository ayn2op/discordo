package config

import (
	_ "embed"
	"log"
	"os"
	"path/filepath"

	"github.com/robertkrimen/otto"
)

//go:embed config.js
var cfg []byte

type Config struct {
	VM *otto.Otto
}

func New() *Config {
	return &Config{
		VM: otto.New(),
	}
}

func (c *Config) Load(path string) error {
	// Create the configuration directory if it does not exist already, recursively.
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	// Create a new configuration file if it does not exist already; otherwise, open the existing file with read-write flag.
	f, err := os.OpenFile(filepath.Join(path, "config.js"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	var src interface{}
	// If the configuration file is empty, that is, its size is zero, write the default configuration to the file.
	if fi.Size() == 0 {
		src = cfg
		f.Write(cfg)
		f.Sync()
	} else {
		src = cfg
	}

	_, err = c.VM.Run(src)
	return err
}

func (c *Config) Value(name string, o *otto.Object) otto.Value {
	var v otto.Value
	if o != nil {
		v, _ = o.Get(name)
	} else {
		v, _ = c.VM.Get(name)
	}

	return v
}

func (c *Config) Bool(name string, o *otto.Object) bool {
	b, _ := c.Value(name, o).ToBoolean()
	return b
}

func (c *Config) Int(name string, o *otto.Object) int64 {
	i, _ := c.Value(name, o).ToInteger()
	return i
}

func (c *Config) String(name string, o *otto.Object) string {
	s, _ := c.Value(name, o).ToString()
	return s
}

func (c *Config) Object(name string, o *otto.Object) *otto.Object {
	v := c.Value(name, o)
	return v.Object()
}

func DefaultDirPath() string {
	path, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(path, "discordo")
}
