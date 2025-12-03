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
const KeybindTomlRegex = `^(?:(?:(Ctrl)|(Alt)|(Shift))\+){0,3}(?:Rune\[)?(.+?)\]?$`

// = 4 capture groups: Ctrl, Alt, Shift, Key

type statusBar struct {
	*tview.TextView
	cfg *config.Config

	keybindRegex *regexp.Regexp
}

func newStatusBar(cfg *config.Config) *statusBar {
	sb := &statusBar{
		TextView:     tview.NewTextView(),
		cfg:          cfg,
		keybindRegex: regexp.MustCompile(KeybindTomlRegex),
	}

	sb.Box = ui.ConfigureBox(sb.Box, &cfg.Theme)
	sb.Box.
		SetBorders(tview.BordersNone).
		SetBlurFunc(nil).
		SetFocusFunc(nil)
	sb.
		SetDynamicColors(true).
		SetWrap(false).
		SetScrollable(false)

	return sb
}

func (sb *statusBar) Update(app *application) {
	if app.chatView == nil || sb.cfg == nil {
		return
	}
	cfg := sb.cfg

	var buffer []string

	var f = app.GetFocus()
	switch f {
	case app.chatView.guildsTree:
		buffer = append(buffer, sb.fmtNavigationKeybinds(cfg.Keys.GuildsTree.NavigationKeys)...)

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
			buffer = append(buffer,
				sb.fmtKeybind(cfg.Keys.GuildsTree.SelectCurrent, label),
			)
		}

	case app.chatView.messagesList:
		buffer = append(buffer, sb.fmtNavigationKeybinds(cfg.Keys.MessagesList.NavigationKeys)...)
		if msg, err := app.chatView.messagesList.selectedMessage(); err == nil {
			buffer = append(buffer,
				sb.fmtKeybind(cfg.Keys.MessagesList.Reply, "reply"),
				sb.fmtKeybind(cfg.Keys.MessagesList.ReplyMention, "@reply"),
			)
			if len(ui.ExtractURLs(msg.Content))+len(msg.Attachments) > 0 {
				buffer = append(buffer,
					sb.fmtKeybind(cfg.Keys.MessagesList.Open, "open"),
				)
			}
			if ref := msg.ReferencedMessage; ref != nil {
				buffer = append(buffer,
					sb.fmtKeybind(cfg.Keys.MessagesList.SelectReply, "goto OP"),
				)
			}
			me, err := discordState.Cabinet.Me()
			if err == nil && me.ID == msg.Author.ID {
				buffer = append(buffer,
					sb.fmtKeybind(cfg.Keys.MessagesList.Edit, "edit"),
					sb.fmtKeybind(cfg.Keys.MessagesList.DeleteConfirm, "delete"),
				)
			}
		}

	case app.chatView.messageInput:
		if app.chatView.GetVisibile(mentionsListPageName) {
			buffer = append(buffer,
				sb.fmtKeybind(cfg.Keys.MessageInput.TabComplete, "accept"),
				sb.fmtKeybind(cfg.Keys.MessageInput.Cancel, "dismiss"),
				sb.fmtKeybind(cfg.Keys.MentionsList.Up, "up"),
				sb.fmtKeybind(cfg.Keys.MentionsList.Down, "down"),
			)
		} else {
			buffer = append(buffer,
				sb.fmtKeybind(cfg.Keys.MessageInput.Send, "send"),
				sb.fmtKeybind(cfg.Keys.MessageInput.Cancel, "clear"),
				sb.fmtKeybind(cfg.Keys.MessageInput.OpenFilePicker, "attach"),
				sb.fmtKeybind(cfg.Keys.MessageInput.Paste, "paste"),
			)
		}

	default:
		// generic tview keybinds
		if app.chatView.GetPageCount() > 1 {
			buffer = append(buffer,
				sb.fmtKeybind("↑→↓←", "navigate"), // hardcoded @ tview.Modal.AddButtons()
				sb.fmtKeybind("Enter", "confirm"), // hardcoded @ tview.Button.InputHandler()
				sb.fmtKeybind("Esc", "cancel"),    // hardcoded @ tview.Button.InputHandler()
			)
		} else {
			// unknown event or mouse controls being difficult
			return
		}

	}
	sb.SetText(strings.Join(buffer, KeybindSeparator))

}

func (sb *statusBar) fmtKeybind(keybind string, label string) string {
	parts := make([]string, 0, 5)

	if matches := sb.keybindRegex.FindStringSubmatchIndex(keybind); len(matches) == 10 {
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

func (sb *statusBar) fmtNavigationKeybinds(navigationKeys config.NavigationKeys) []string {
	return []string{
		sb.fmtKeybind(navigationKeys.SelectPrevious, "prev"),
		sb.fmtKeybind(navigationKeys.SelectNext, "next"),
		sb.fmtKeybind(navigationKeys.SelectFirst, "first"),
		sb.fmtKeybind(navigationKeys.SelectLast, "last")}
}
