package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var messageInputFieldBackgroundColor = tcell.GetColor("#1C1E26")
var messageInputFieldPlaceholderTextColor = tcell.ColorWhite

func NewMessageInputField(onMessageInputFieldDone func(key tcell.Key)) *tview.InputField {
	messageInputField := tview.NewInputField().
		SetPlaceholder("Message...").
		SetFieldWidth(0).
		SetDoneFunc(onMessageInputFieldDone)
	messageInputField.
		SetFieldBackgroundColor(messageInputFieldBackgroundColor).
		SetPlaceholderTextColor(messageInputFieldPlaceholderTextColor).
		SetBackgroundColor(messageInputFieldBackgroundColor).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return messageInputField
}
