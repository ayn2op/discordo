package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewMessageInputField(onMessageInputFieldInputCapture func(*tcell.EventKey) *tcell.EventKey, theme *util.Theme) (i *tview.InputField) {
	i = tview.NewInputField()
	i.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.GetColor(theme.InputFieldBackground)).
		SetBackgroundColor(tcell.GetColor(theme.InputFieldBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetInputCapture(onMessageInputFieldInputCapture)

	return
}
