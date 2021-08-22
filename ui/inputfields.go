package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

// NewMessageInputField creates and returns a new message inputfield.
func NewMessageInputField(onMessageInputFieldInputCapture func(*tcell.EventKey) *tcell.EventKey, t *util.Theme) *tview.InputField {
	i := tview.NewInputField()
	i.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.GetColor(t.Background)).
		SetBackgroundColor(tcell.GetColor(t.Background)).
		SetInputCapture(onMessageInputFieldInputCapture).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return i
}
