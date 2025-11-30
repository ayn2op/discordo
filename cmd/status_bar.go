package cmd

import (
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
)

type statusBar struct {
	*tview.TextView
	cfg *config.Config
}

func newStatusBar(cfg *config.Config) *statusBar {
	sb := &statusBar{
		TextView: tview.NewTextView(),
		cfg:      cfg,
	}

	sb.Box = ui.ConfigureBox(sb.Box, &cfg.Theme)
	sb.Box.
		SetTitleAlignment(tview.AlignmentLeft).
		SetBorders(tview.BordersNone).
		SetBlurFunc(nil).
		SetFocusFunc(nil)
	sb.
		SetWrap(false).
		SetScrollable(false).
		SetTitle("")

	return sb
}

// TODO)) ideally these are NOT hardcoded rofl
func (sb *statusBar) update(app *application) {
	if app.chatView != nil {
		helpString := ""
		var f = app.GetFocus()
		switch f {
		case app.chatView.guildsTree:
			helpString += "k prev j next g first G last RTN select"
			// TODO)) expand/collapse
		case app.chatView.messagesList:
			helpString += "k prev j next g first G last"
			msg, err := app.chatView.messagesList.selectedMessage()
			if err != nil {
				// noop i think?
			} else {
				helpString += " r reply R @reply"
				urls := ui.ExtractURLs(msg.Content)
				if len(urls)+len(msg.Attachments) > 0 {
					helpString += " o open"
				}
				if ref := msg.ReferencedMessage; ref != nil {
					helpString += " s goto OP"
				}
				// TODO)) edit/delete own messages, attachments list
			}
		case app.chatView.messageInput:
			// TODO))
			helpString += "RTN send ALT-RTN newline ESC clear CTRL-\\ attach"
		default:
			// TODO)) mouse input seems to cause this case, not sure of a solution :(
			helpString += " (mouse controls dont play nice with status bar atm.)"
		}
		sb.setText(helpString)
	}
}

func (sb *statusBar) setText(t string) {
	sb.SetText(t)
}
