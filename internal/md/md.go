package md

import "github.com/yuin/goldmark"

func New() goldmark.Markdown {
	return goldmark.New(goldmark.WithRenderer(newRenderer()))
}
