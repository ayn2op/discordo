package cmd

import (
	"fmt"
	"strings"

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
		SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(false).
		SetTitle("")

	return sb
}

func (sb *statusBar) update(app *application) {
	if app.chatView != nil {
		var keybinds []string
		cfg := app.cfg

		var f = app.GetFocus()
		switch f {
		case app.chatView.guildsTree:
			keybinds = append(keybinds,
				fmtKeybind(cfg.Keys.GuildsTree.SelectPrevious, "prev"),
				fmtKeybind(cfg.Keys.GuildsTree.SelectNext, "next"),
				fmtKeybind(cfg.Keys.GuildsTree.SelectFirst, "first"),
				fmtKeybind(cfg.Keys.GuildsTree.SelectLast, "last"),
			)
			// TODO)) expand/collapse
			// cfg.Keys.GuildsTree.SelectCurrent, "select"
			// cfg.Keys.GuildsTree.SelectCurrent, "expand"
			// cfg.Keys.GuildsTree.SelectCurrent, "collapse"
		case app.chatView.messagesList:
			keybinds = append(keybinds,
				fmtKeybind(cfg.Keys.MessagesList.SelectPrevious, "prev"),
				fmtKeybind(cfg.Keys.MessagesList.SelectNext, "next"),
				fmtKeybind(cfg.Keys.MessagesList.SelectFirst, "first"),
				fmtKeybind(cfg.Keys.MessagesList.SelectLast, "last"),
			)
			msg, err := app.chatView.messagesList.selectedMessage()
			if err != nil {
				// noop i think?
			} else {
				keybinds = append(keybinds,
					fmtKeybind(cfg.Keys.MessagesList.Reply, "reply"),
					fmtKeybind(cfg.Keys.MessagesList.ReplyMention, "@reply"),
				)
				urls := ui.ExtractURLs(msg.Content)
				if len(urls)+len(msg.Attachments) > 0 {
					keybinds = append(keybinds,
						fmtKeybind(cfg.Keys.MessagesList.Open, "open"),
					)
				}
				if ref := msg.ReferencedMessage; ref != nil {
					keybinds = append(keybinds,
						fmtKeybind(cfg.Keys.MessagesList.SelectReply, "goto OP"),
					)
				}
				// TODO)) edit/delete own messages
				// cfg.Keys.MessagesList.Edit, "edit"
				// cfg.Keys.MessagesList.Delete, "delete"
				// cfg.Keys.MessagesList.DeleteConfirm, "confirm delete"
				// TODO)) attachments list
			}
		case app.chatView.messageInput:
			// TODO))
			keybinds = append(keybinds,
				fmtKeybind(cfg.Keys.MessageInput.Send, "send"),
				fmtKeybind(cfg.Keys.MessageInput.Cancel, "clear"),
				fmtKeybind(cfg.Keys.MessageInput.OpenFilePicker, "attach"),
			)
		default:
			// TODO)) mouse input seems to cause this case, not sure of a solution :(
			keybinds = append(keybinds,
				"(status bar not playing nice with mouse input atm.)",
			)
		}
		// TODO)) expose separator to theme?
		sb.setText(strings.Join(keybinds, "  "))
	}
}

func fmtKeybind(keybind string, name string) string {
	// this is seriously ugly and probably not performance friendly
	key := keybind
	if strings.HasPrefix(key, "Rune[") {
		key = fmt.Sprintf("%c", []rune(keybind)[5]) // bro
	}
	key = strings.Replace(key, "Ctrl+", "^", 1)
	key = strings.Replace(key, "Enter", "CR", 1)
	key = strings.Replace(key, "Esc", "ESC", 1)
	// TODO)) expose format to theme?
	return fmt.Sprintf("[::r][::b] %v [::B][::R] %v", key, name)
}

func (sb *statusBar) setText(t string) {
	sb.SetText(t)
}
