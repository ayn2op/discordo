package main

import (
	"github.com/rivo/tview"
)

type MessageInput struct {
	*tview.InputField
}

func newMessageInput() *MessageInput {
	mi := &MessageInput{
		InputField: tview.NewInputField(),
	}

	mi.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	mi.SetBorder(cfg.Theme.MessageInput.Border)

	padding := cfg.Theme.MessageInput.BorderPadding
	mi.SetBorderPadding(padding[0], padding[1], padding[2], padding[3])

	return mi
}
