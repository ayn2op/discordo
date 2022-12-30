package main

import "github.com/rivo/tview"

type MessageInput struct {
	*tview.InputField
}

func newMessageInput() *MessageInput {
	mi := &MessageInput{
		InputField: tview.NewInputField(),
	}

	mi.SetBorder(true)
	mi.SetBorderPadding(cfg.BorderPadding())

	return mi
}
