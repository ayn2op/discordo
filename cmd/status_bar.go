package cmd

import (
	"fmt"
	"reflect"

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
		SetRegions(true).
		SetWrap(false).
		SetTitle("Status Bar!")

	return sb
}

func (sb *statusBar) update(app *application) {
	if app.chatView != nil {
		var f = app.GetFocus()
		switch f {
		// ideally these are NOT hardcoded rofl
		case app.chatView.guildsTree:
			sb.setText("k prev j next g first G last RTN select")
		case app.chatView.messagesList:
			sb.setText("k prev j next g first G last r reply R @reply")
		case app.chatView.messageInput:
			sb.setText("RTN send ALT-RTN newline ESC clear CTRL-\\ attach")
		default:
			// mouse input seems to cause this case, not sure of a solution :(
			sb.setText(fmt.Sprint(reflect.TypeOf(f)))
		}
	}
}

func (sb *statusBar) setText(t string) {
	sb.SetTitle(t)
}
