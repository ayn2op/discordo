package commandslist

import "github.com/ayn2op/tview"

type Item struct {
	Name        string
	Usage       string
	Description string
	Source      string
	InsertText  string
	DisplayText string
	Style       tview.Style
}
