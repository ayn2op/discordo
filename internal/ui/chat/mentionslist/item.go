package mentionslist

import "github.com/gdamore/tcell/v3"

type Item struct {
	InsertText  string
	DisplayText string
	Style       tcell.Style
}
