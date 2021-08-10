package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewMessagesTextView(app *tview.Application, theme *util.Theme) *tview.TextView {
	messagesTextView := tview.NewTextView()

	messagesTextView.
		SetDynamicColors(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetChangedFunc(onMessagesTextViewChanged(app)).
		SetBackgroundColor(tcell.GetColor(theme.TextViewBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return messagesTextView
}

func onMessagesTextViewChanged(app *tview.Application) func() {
	return func() {
		app.Draw()
	}
}
