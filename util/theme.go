package util

import (
	"encoding/json"
	"os"
)

type Theme struct {
	DropDownBackground              string `json:"dropdown.background"`
	DropDownForeground              string `json:"dropdown.foreground"`
	InputFieldBackground            string `json:"inputField.background"`
	InputFieldForeground            string `json:"inputField.foreground"`
	InputFieldPlaceholderForeground string `json:"inputField.placeholderTextForeground"`
	ListBackground                  string `json:"list.background"`
	ListMainTextForeground          string `json:"list.mainTextForeground"`
	ListSelectedForeground          string `json:"list.selectedTextForeground"`
	TextViewBackground              string `json:"textview.background"`
	TextViewForeground              string `json:"textview.foreground"`
}

func NewTheme() *Theme {
	var theme Theme
	theme.TextViewBackground = "#2E3440"
	theme.TextViewForeground = "#D8DEE9"
	theme.ListBackground = "#2E3440"
	theme.ListMainTextForeground = "#4C566A"
	theme.ListSelectedForeground = "#ECEFF4"
	theme.InputFieldBackground = "#3B4252"
	theme.InputFieldForeground = "#D8DEE9"
	theme.InputFieldPlaceholderForeground = "#D8DEE9"
	theme.DropDownBackground = "#3B4252"
	theme.DropDownForeground = "#D8DEE9"

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
