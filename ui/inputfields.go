package ui

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewMessageInputField(s *session.Session, c discord.Channel, theme *util.Theme) *tview.InputField {
	i := tview.NewInputField()
	i.
		SetPlaceholder("Message...").
		SetFieldWidth(0).
		SetDoneFunc(onMessageInputFieldDone(i, s, c)).
		SetFieldBackgroundColor(tcell.GetColor(theme.InputFieldBackground)).
		SetBackgroundColor(tcell.GetColor(theme.InputFieldBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1).
		SetInputCapture(onMessageInputFieldInputCapture(i))

	return i
}

func onMessageInputFieldDone(i *tview.InputField, s *session.Session, c discord.Channel) func(k tcell.Key) {
	return func(k tcell.Key) {
		if k == tcell.KeyEnter {
			t := strings.TrimSpace(i.GetText())
			if t == "" {
				return
			}

			i.SetText("")
			s.SendMessage(c.ID, t)
		}
	}
}

func onMessageInputFieldInputCapture(i *tview.InputField) func(event *tcell.EventKey) *tcell.EventKey {
	return func(event *tcell.EventKey) *tcell.EventKey {
		if event.Key() == tcell.KeyCtrlV {
			text, _ := clipboard.ReadAll()
			text = i.GetText() + text
			i.SetText(text)
		}

		return event
	}
}
