package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func NewMessagesTextView() *tview.TextView {
	t := tview.NewTextView()
	t.SetBorder(true)
	t.SetBorderPadding(0, 0, 1, 1)

	return t
}

func NewMessageInputField(a *App) *tview.InputField {
	i := tview.NewInputField()
	i.SetPlaceholder("Message...")
	i.SetBorder(true)
	i.SetBorderPadding(0, 0, 1, 1)

	i.SetFieldStyle(
		tcell.StyleDefault.
			Background(tview.Styles.PrimitiveBackgroundColor).
			Foreground(tcell.GetColor(a.config.Theme.MessageInputField.FieldForeground)),
	)
	i.SetPlaceholderStyle(
		tcell.StyleDefault.
			Background(tview.Styles.PrimitiveBackgroundColor).
			Foreground(tcell.GetColor(a.config.Theme.MessageInputField.PlaceholderForeground)),
	)

	return i
}
