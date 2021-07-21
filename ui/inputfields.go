package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewMessageInputField(onMessageInputFieldDone func(key tcell.Key)) (messageInputField *tview.InputField) {
	messageInputField = tview.NewInputField().
		SetPlaceholder("Message...").
		SetFieldWidth(0).
		SetDoneFunc(onMessageInputFieldDone)
	messageInputField.
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetPlaceholderTextColor(tcell.ColorDarkGray).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}
