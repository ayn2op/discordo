package util

import (
	"encoding/json"
	"os"
)

type Theme struct {
	DropDownBackground   string `json:"dropdown.background"`
	TreeViewBackground   string `json:"treeview.background"`
	TextViewBackground   string `json:"textview.background"`
	InputFieldBackground string `json:"inputField.background"`
}

type Config struct {
	GetMessagesLimit uint   `json:"getMessagesLimit"`
	Theme            *Theme `json:"theme"`
}

func NewConfig() *Config {
	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	var c Config = Config{
		GetMessagesLimit: 50,
		Theme:            &Theme{},
	}
	configPath := userHomeDir + "/.config/discordo/config.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &c
	}

	d, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(d, &c); err != nil {
		panic(err)
	}

	return &c
}
