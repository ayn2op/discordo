package cmd

import (
	"fmt"
	"regexp"
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
				var label string

				if len(node.GetChildren()) != 0 {
					if node.IsExpanded() {
						label = "collapse"
					} else {
						label = "expand"
					}
				} else {
					switch node.GetReference().(type) {
					case discord.ChannelID:
						label = "select"
					default:
						label = "expand"
					}
				}
				keybinds = append(keybinds,
					fmtKeybind(cfg.Keys.GuildsTree.SelectCurrent, label),
				)
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

func fmtKeybind(keybind string, label string) string {
	r := regexp.MustCompile(`^(?:(?:(Ctrl)|(Alt)|(Shift))\+){0,3}(?:Rune\[)?(.+?)\]?$`)
	// = 4 capture groups: Ctrl, Alt, Shift, Key
	parts := make([]string, 0, 5)

	if matches := r.FindStringSubmatchIndex(keybind); len(matches) == 10 {
		// matches[0] to matches[1] is the whole match
		if matches[2] != matches[3] {
			parts = append(parts, "^") // Ctrl
		}
		if matches[4] != matches[5] {
			parts = append(parts, "ALT-") // Alt
		}
		if matches[6] != matches[7] {
			parts = append(parts, "SHFT-") // Shift
		}
		key := string([]byte(keybind)[matches[8]:matches[9]])
		switch string(key) {
		case "Enter":
			parts = append(parts, "CR")
		case " ":
			parts = append(parts, "SPACE")
		default:
			if matches[9] == matches[8]+1 {
				parts = append(parts, string(key))
			} else {
				parts = append(parts, strings.ToUpper(key))
			}
		}
	}

	resultString := strings.Join(parts, "")
	return fmt.Sprintf(KeybindFormat, resultString, label)
}

func fmtNavigationKeybinds(navigationKeys config.NavigationKeys) []string {
	return []string{
		fmtKeybind(navigationKeys.SelectPrevious, "prev"),
		fmtKeybind(navigationKeys.SelectNext, "next"),
		fmtKeybind(navigationKeys.SelectFirst, "first"),
		fmtKeybind(navigationKeys.SelectLast, "last")}
}
