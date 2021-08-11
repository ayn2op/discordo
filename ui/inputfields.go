package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewMessageInputField(onMessageInputFieldInputCapture func(event *tcell.EventKey) *tcell.EventKey, s *session.Session, c discord.Channel, theme *util.Theme) *tview.InputField {
	i := tview.NewInputField()
	i.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tcell.GetColor(theme.InputFieldBackground)).
		SetBackgroundColor(tcell.GetColor(theme.InputFieldBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetInputCapture(onMessageInputFieldInputCapture)

	return i
}
