package chat

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/layers"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v3"
)

const (
	userContextMenuLayerName = "userContextMenu"
	noteFormLayerName        = "noteForm"
	profileLayerName         = "profile"
)

type userContextMenuItem struct {
	key    rune
	label  string
	action func()
}

type userContextMenu struct {
	*tview.List
	items         []userContextMenuItem
	chatView      *View
	cfg           *config.Config
	previousFocus tview.Primitive
}

var _ help.KeyMap = (*userContextMenu)(nil)

func newUserContextMenu(cfg *config.Config, chatView *View) *userContextMenu {
	m := &userContextMenu{
		List:     tview.NewList(),
		cfg:      cfg,
		chatView: chatView,
	}

	m.Box = ui.ConfigureBox(m.Box, &cfg.Theme)
	m.SetSnapToItems(true)
	m.SetBuilder(m.buildItem)
	return m
}

func (m *userContextMenu) buildItem(index int, _ int) tview.ListItem {
	if index < 0 || index >= len(m.items) {
		return nil
	}

	item := m.items[index]
	var baseStyle tcell.Style
	keyStyle := baseStyle.Foreground(tcell.ColorGreen).Underline(true)

	// Find the hotkey letter within the label and style it inline.
	keyStr := strings.ToUpper(string(item.key))
	idx := strings.Index(strings.ToUpper(item.label), keyStr)

	var segments []tview.Segment
	if idx >= 0 {
		if idx > 0 {
			segments = append(segments, tview.NewSegment(item.label[:idx], baseStyle))
		}
		segments = append(segments, tview.NewSegment(item.label[idx:idx+len(keyStr)], keyStyle))
		if idx+len(keyStr) < len(item.label) {
			segments = append(segments, tview.NewSegment(item.label[idx+len(keyStr):], baseStyle))
		}
	} else {
		segments = append(segments, tview.NewSegment(item.label, baseStyle))
	}

	line := tview.NewLine(segments...)

	return tview.NewTextView().
		SetScrollable(false).
		SetWrap(false).
		SetWordWrap(false).
		SetLines([]tview.Line{line})
}

func (m *userContextMenu) build(user discord.User, guildID discord.GuildID) {
	m.items = nil

	me, _ := m.chatView.state.Cabinet.Me()
	isSelf := me.ID == user.ID

	title := user.DisplayOrUsername()
	m.SetTitle(title)

	// Profile — always shown
	m.items = append(m.items, userContextMenuItem{
		key:   'P',
		label: "Profile",
		action: func() {
			m.showProfile(user, guildID)
		},
	})

	if !isSelf {
		rel := m.chatView.state.RelationshipState.Relationship(user.ID)

		// Show "Message" only if we're friends or have an existing DM channel,
		// since first-time DMs to non-friends typically hit a captcha wall.
		if rel == discord.FriendRelationship || m.hasExistingDM(user.ID) {
			m.items = append(m.items, userContextMenuItem{
				key:   'M',
				label: "Message",
				action: func() {
					go m.openDM(user.ID)
				},
			})
		}
	}

	// Add Note — always shown
	m.items = append(m.items, userContextMenuItem{
		key:   'N',
		label: "Note",
		action: func() {
			m.showNoteForm(user.ID)
		},
	})

	m.SetCursor(-1)
	m.MarkDirty()
}

// showAt displays the menu anchored near the given screen position. The menu
// is placed above anchorY when there is room, otherwise below it.
func (m *userContextMenu) showAt(anchorX, anchorY int) {
	m.previousFocus = m.chatView.app.GetFocus()

	// Compute menu dimensions.
	menuWidth := 0
	for _, item := range m.items {
		// "[X] Label" = 4 + len(label)
		w := 4 + len(item.label)
		if w > menuWidth {
			menuWidth = w
		}
	}
	// Account for border + padding.
	menuWidth += 4
	menuHeight := len(m.items) + 2

	_, _, screenWidth, screenHeight := m.chatView.GetRect()

	x := anchorX
	if x+menuWidth > screenWidth {
		x = screenWidth - menuWidth
	}
	if x < 0 {
		x = 0
	}

	y := anchorY - menuHeight
	if y < 0 {
		y = anchorY
	}
	if y+menuHeight > screenHeight {
		y = screenHeight - menuHeight
	}
	if y < 0 {
		y = 0
	}

	m.SetRect(x, y, menuWidth, menuHeight)

	m.chatView.
		AddLayer(
			m,
			layers.WithName(userContextMenuLayerName),
			layers.WithVisible(true),
			layers.WithOverlay(),
		).
		SendToFront(userContextMenuLayerName)
	m.chatView.app.SetFocus(m)
}

func (m *userContextMenu) hide() {
	m.chatView.RemoveLayer(userContextMenuLayerName)
	if m.previousFocus != nil {
		m.chatView.app.SetFocus(m.previousFocus)
	}
}

func (m *userContextMenu) executeByKey(s string) {
	s = strings.ToUpper(s)
	for _, item := range m.items {
		if string(item.key) == s {
			action := item.action
			m.hide()
			action()
			return
		}
	}
}

func (m *userContextMenu) handleInput(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyEscape {
		m.hide()
		return nil
	}

	if event.Key() == tcell.KeyRune {
		m.executeByKey(event.Str())
		return nil
	}

	return nil
}

func (m *userContextMenu) InputHandler(event *tcell.EventKey, setFocus func(p tview.Primitive)) {
	event = m.handleInput(event)
	if event == nil {
		return
	}
	m.List.InputHandler(event, setFocus)
}

func (m *userContextMenu) hasExistingDM(userID discord.UserID) bool {
	channels, err := m.chatView.state.PrivateChannels()
	if err != nil {
		return false
	}
	for _, ch := range channels {
		if ch.Type == discord.DirectMessage {
			for _, r := range ch.DMRecipients {
				if r.ID == userID {
					return true
				}
			}
		}
	}
	return false
}

// Action implementations

func (m *userContextMenu) openDM(userID discord.UserID) {
	channel, err := m.chatView.state.CreatePrivateChannel(userID)
	if err != nil {
		slog.Error("failed to create private channel", "user_id", userID, "err", err)
		return
	}

	m.chatView.app.QueueUpdateDraw(func() {
		gt := m.chatView.guildsTree

		// Ensure the DM root node is expanded so channel nodes exist.
		if dmRoot := gt.findNodeByReference(dmNode{}); dmRoot != nil {
			if len(dmRoot.GetChildren()) == 0 {
				gt.onSelected(dmRoot)
			}
		}

		// If the channel node doesn't exist yet (newly created DM), add it.
		node := gt.findNodeByReference(channel.ID)
		if node == nil {
			if dmRoot := gt.findNodeByReference(dmNode{}); dmRoot != nil {
				gt.createChannelNode(dmRoot, *channel)
				node = gt.findNodeByReference(channel.ID)
			}
		}

		if node == nil {
			slog.Error("failed to find DM channel node", "channel_id", channel.ID)
			return
		}

		gt.expandPathToNode(node)
		gt.SetCurrentNode(node)
		gt.onSelected(node)
	})
}

func (m *userContextMenu) showNoteForm(userID discord.UserID) {
	previousFocus := m.chatView.app.GetFocus()
	currentNote := m.chatView.state.NoteState.Note(userID)

	form := tview.NewForm()
	form.AddInputField("Note", currentNote, 40, nil)
	form.AddButton("Save", func() {
		noteField, _ := form.GetFormItemByLabel("Note").(*tview.InputField)
		note := noteField.GetText()
		go func() {
			if err := m.chatView.state.SetNote(userID, note); err != nil {
				slog.Error("failed to set note", "user_id", userID, "err", err)
			}
		}()
		m.chatView.RemoveLayer(noteFormLayerName)
		m.chatView.app.SetFocus(previousFocus)
	})
	form.AddButton("Cancel", func() {
		m.chatView.RemoveLayer(noteFormLayerName)
		m.chatView.app.SetFocus(previousFocus)
	})
	form.SetCancelFunc(func() {
		m.chatView.RemoveLayer(noteFormLayerName)
		m.chatView.app.SetFocus(previousFocus)
	})
	form.SetTitle("Set Note")
	form.SetBorders(tview.BordersAll)

	m.chatView.
		AddLayer(
			ui.Centered(form, 50, 7),
			layers.WithName(noteFormLayerName),
			layers.WithResize(true),
			layers.WithVisible(true),
			layers.WithOverlay(),
		).
		SendToFront(noteFormLayerName)
	m.chatView.app.SetFocus(form)
}

func (m *userContextMenu) showProfile(user discord.User, guildID discord.GuildID) {
	previousFocus := m.chatView.app.GetFocus()

	var b strings.Builder
	fmt.Fprintf(&b, "Display Name: %s\n", user.DisplayOrUsername())
	fmt.Fprintf(&b, "Username:     %s\n", user.Username)
	if user.Discriminator != "0" && user.Discriminator != "" {
		fmt.Fprintf(&b, "Discriminator: #%s\n", user.Discriminator)
	}
	fmt.Fprintf(&b, "ID:           %s\n", user.ID)
	fmt.Fprintf(&b, "Created:      %s\n", user.CreatedAt().Format("January 2, 2006"))
	if user.Bot {
		b.WriteString("Bot:          Yes\n")
	}

	if guildID.IsValid() {
		if member, err := m.chatView.state.Cabinet.Member(guildID, user.ID); err == nil {
			if member.Nick != "" {
				fmt.Fprintf(&b, "Nickname:     %s\n", member.Nick)
			}
			if member.Joined.IsValid() {
				fmt.Fprintf(&b, "Joined:       %s\n", member.Joined.Time().Format("January 2, 2006"))
			}
			if len(member.RoleIDs) > 0 {
				var roles []string
				for _, roleID := range member.RoleIDs {
					if role, err := m.chatView.state.Cabinet.Role(guildID, roleID); err == nil {
						roles = append(roles, role.Name)
					}
				}
				if len(roles) > 0 {
					fmt.Fprintf(&b, "Roles:        %s\n", strings.Join(roles, ", "))
				}
			}
		}
	}

	if note := m.chatView.state.NoteState.Note(user.ID); note != "" {
		fmt.Fprintf(&b, "Note:         %s\n", note)
	}

	tv := tview.NewTextView().
		SetText(b.String()).
		SetScrollable(true).
		SetWrap(true).
		SetWordWrap(true)
	tv.SetTitle("Profile — " + user.DisplayOrUsername())
	tv.SetBorders(tview.BordersAll)

	pv := &profileView{
		TextView: tv,
		close: func() {
			m.chatView.RemoveLayer(profileLayerName)
			m.chatView.app.SetFocus(previousFocus)
		},
	}

	m.chatView.
		AddLayer(
			ui.Centered(pv, 60, 16),
			layers.WithName(profileLayerName),
			layers.WithResize(true),
			layers.WithVisible(true),
			layers.WithOverlay(),
		).
		SendToFront(profileLayerName)
	m.chatView.app.SetFocus(pv)
}

// profileView wraps a TextView to intercept Escape for closing.
type profileView struct {
	*tview.TextView
	close func()
}

func (p *profileView) InputHandler(event *tcell.EventKey, setFocus func(tview.Primitive)) {
	if event.Key() == tcell.KeyEscape {
		p.close()
		return
	}
	p.TextView.InputHandler(event, setFocus)
}

// Help bar integration

func (m *userContextMenu) ShortHelp() []keybind.Keybind {
	binds := make([]keybind.Keybind, 0, len(m.items)+1)
	for _, item := range m.items {
		binds = append(binds, keybind.NewKeybind(
			keybind.WithKeys(string(item.key)),
			keybind.WithHelp(strings.ToLower(string(item.key)), item.label),
		))
	}
	binds = append(binds, keybind.NewKeybind(
		keybind.WithKeys("esc"),
		keybind.WithHelp("esc", "cancel"),
	))
	return binds
}

func (m *userContextMenu) FullHelp() [][]keybind.Keybind {
	return [][]keybind.Keybind{m.ShortHelp()}
}
