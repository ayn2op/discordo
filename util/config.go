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

	DropDownForeground              string `json:"dropdown.foreground"`
	TextViewForeground              string `json:"textview.foreground"`
	TreeNodeForeground              string `json:"treenode.foreground"`
	InputFieldForeground            string `json:"inputField.foreground"`
	InputFieldPlaceholderForeground string `json:"inputField.placeholderTextForeground"`
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

	var config Config = Config{
		GetMessagesLimit: 50,
		Theme:            &Theme{},
	}
	configPath := userHomeDir + "/.config/discordo/config.json"
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &config
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(data, &config); err != nil {
		panic(err)
	}

	return &config
}
