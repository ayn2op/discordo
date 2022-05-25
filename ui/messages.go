package ui

import "github.com/rivo/tview"

func NewMessagesTextView() *tview.TextView {
	t := tview.NewTextView()
	t.SetBorder(true)
	t.SetBorderPadding(0, 0, 1, 1)

	return t
}

func NewMessageInputField() *tview.InputField {
	i := tview.NewInputField()
	i.SetPlaceholder("Message...")
	i.SetBorder(true)
	i.SetBorderPadding(0, 0, 1, 1)

	return i
}
