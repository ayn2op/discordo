package cmd

import (
	"fmt"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
)

const KeybindFormat = "[::r][::b] %s [::B][::R] %s"
const KeybindSeparator = "  "

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
			keybinds = append(keybinds, fmtNavigationKeybinds(cfg.Keys.GuildsTree.NavigationKeys)...)
			node := app.chatView.guildsTree.GetCurrentNode()

			if node != nil {
				switch node.GetReference().(type) {
				case discord.ChannelID:
					keybinds = append(keybinds,
						fmtKeybind(cfg.Keys.GuildsTree.SelectCurrent, "select"),
					)
				default:
					if len(node.GetChildren()) == 0 || !node.IsExpanded() {
						keybinds = append(keybinds,
							fmtKeybind(cfg.Keys.GuildsTree.SelectCurrent, "expand"),
						)
					} else {
						keybinds = append(keybinds,
							fmtKeybind(cfg.Keys.GuildsTree.SelectCurrent, "collapse"),
						)
					}
				}
			}

		case app.chatView.messagesList:
			keybinds = append(keybinds, fmtNavigationKeybinds(cfg.Keys.MessagesList.NavigationKeys)...)
			if msg, err := app.chatView.messagesList.selectedMessage(); err == nil {
				keybinds = append(keybinds,
					fmtKeybind(cfg.Keys.MessagesList.Reply, "reply"),
					fmtKeybind(cfg.Keys.MessagesList.ReplyMention, "@reply"),
				)
				if len(ui.ExtractURLs(msg.Content))+len(msg.Attachments) > 0 {
					keybinds = append(keybinds,
						fmtKeybind(cfg.Keys.MessagesList.Open, "open"),
					)
				}
				if ref := msg.ReferencedMessage; ref != nil {
					keybinds = append(keybinds,
						fmtKeybind(cfg.Keys.MessagesList.SelectReply, "goto OP"),
					)
				}
				me, err := discordState.Cabinet.Me()
				if err == nil && me.ID == msg.Author.ID {
					keybinds = append(keybinds,
						fmtKeybind(cfg.Keys.MessagesList.Edit, "edit"),
						fmtKeybind(cfg.Keys.MessagesList.DeleteConfirm, "delete"),
					)
				}
			}

		case app.chatView.messageInput:
			if app.chatView.GetVisibile(mentionsListPageName) {
				keybinds = append(keybinds,
					fmtKeybind(cfg.Keys.MessageInput.TabComplete, "accept"),
					fmtKeybind(cfg.Keys.MessageInput.Cancel, "dismiss"),
					fmtKeybind(cfg.Keys.MentionsList.Up, "up"),
					fmtKeybind(cfg.Keys.MentionsList.Down, "down"),
				)
			} else {
				keybinds = append(keybinds,
					fmtKeybind(cfg.Keys.MessageInput.Send, "send"),
					fmtKeybind(cfg.Keys.MessageInput.Cancel, "clear"),
					fmtKeybind(cfg.Keys.MessageInput.OpenFilePicker, "attach"),
					fmtKeybind(cfg.Keys.MessageInput.Paste, "paste"),
				)
			}

		default:
			if app.chatView.GetPageCount() > 1 {
				keybinds = append(keybinds,
					fmtKeybind("↑→↓←", "navigate"),
					fmtKeybind("Enter", "confirm"),
					fmtKeybind("Esc", "cancel"),
				)
			} else {
				// TODO)) mouse input seems to cause this case, not sure of a solution :(
				keybinds = append(keybinds,
					"(status bar not playing nice with mouse input atm.)",
				)
			}
		}

		sb.SetText(strings.Join(keybinds, KeybindSeparator))
	}
}

// this is seriously ugly and probably not performance friendly
func fmtKeybind(keybind string, label string) string {
	key := keybind
	if strings.HasPrefix(key, "Rune[") {
		key = fmt.Sprintf("%c", []rune(keybind)[5]) // bro
	} else {
		key = strings.Replace(key, "Ctrl+", "^", 1)
		key = strings.Replace(key, "Alt+", "ALT-", 1)
		key = strings.Replace(key, "Enter", "CR", 1)
		key = strings.ToUpper(key)
	}

	return fmt.Sprintf("[::r][::b] %v [::B][::R] %v", key, label)
}

func fmtNavigationKeybinds(navigationKeys config.NavigationKeys) []string {
	return []string{
		fmtKeybind(navigationKeys.SelectPrevious, "prev"),
		fmtKeybind(navigationKeys.SelectNext, "next"),
		fmtKeybind(navigationKeys.SelectFirst, "first"),
		fmtKeybind(navigationKeys.SelectLast, "last")}
}
