package ui

import (
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

func NewMessagesTextView(onMessagesTextViewChanged func(), theme *util.Theme) (messagesTextView *tview.TextView) {
	messagesTextView = tview.NewTextView().
		SetDynamicColors(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetChangedFunc(onMessagesTextViewChanged)
	messagesTextView.
		SetTextColor(tcell.GetColor(theme.TextViewForeground)).
		SetBackgroundColor(tcell.GetColor(theme.TextViewBackground)).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 1)

	return
}
