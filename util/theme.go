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

func NewTheme() *Theme {
	var theme Theme
	theme.DropDownBackground = "#3B4252"
	theme.TreeViewBackground = "#282a36"
	theme.TextViewBackground = "#282a36"
	theme.InputFieldBackground = "#3B4252"

	theme.DropDownForeground = "#f8f8f2"
	theme.TextViewForeground = "#f8f8f2"
	theme.TreeNodeForeground = "#8be9fd"
	theme.InputFieldForeground = "#f8f8f2"
	theme.InputFieldPlaceholderForeground = "#6272a4"

	userHomeDir, err := os.UserHomeDir()
	if err != nil {
		panic(err)
	}

	themeFilePath := userHomeDir + "/.config/discordo/theme.json"
	if _, err := os.Stat(themeFilePath); os.IsNotExist(err) {
		return &theme
	}

	data, err := os.ReadFile(themeFilePath)
	if err != nil {
		panic(err)
	}

	if err = json.Unmarshal(data, &theme); err != nil {
		panic(err)
	}

	return &theme
}
