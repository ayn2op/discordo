package emojislist

import "github.com/ayn2op/tview"

type Item struct {
	Name        string
	InsertText  string
	DisplayText string
	Style       tview.Style
}
