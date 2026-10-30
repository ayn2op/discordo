package chat

import (
	"fmt"
	"image"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/images"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg, "")
	return &m
}

// chatArea is the area the whole chat is laid out in to press a key in it, and maxCommandHops bounds how
// many commands one key may set off, so a loop of them fails the test rather than hanging it.
var chatArea = tview.Rectangle{Width: 120, Height: 40}

const maxCommandHops = 16

// press feeds key to the view and returns every message it and its commands produce, as the event loop does.
func press(t *testing.T, m *Model, key tview.Msg) []tview.Msg {
	t.Helper()
	var produced []tview.Msg
	msg := m.View().Handle(key, chatArea)
	for range maxCommandHops {
		if msg == nil {
			return produced
		}

		produced = append(produced, msg)

		var cmd tview.Cmd
		*m, cmd = m.Update(msg)
		if cmd == nil {
			return produced
		}

		msg = cmd()
	}

	t.Fatal("the commands a key set off did not settle")
	return nil
}

// composerSize is the area the composer is drawn in to read what it shows.
var composerSize = tview.Rectangle{Width: 40, Height: 5}

// composerText returns everything the composer draws, including its title.
func composerText(t *testing.T, m *Model) string {
	t.Helper()
	screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: vt.Col(composerSize.Width), Y: vt.Row(composerSize.Height)}))
	if err != nil {
		t.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()

	m.composer.View().Draw(screen, composerSize)

	var drawn strings.Builder
	for y := range composerSize.Height {
		for x := range composerSize.Width {
			str, _, _ := screen.Get(x, y)
			drawn.WriteString(str)
		}
	}
	return drawn.String()
}

func TestModelView(t *testing.T) {
	t.Run("typing in the channels picker", func(t *testing.T) {
		m := newTestModel(t)
		m.openPicker()
		area := tview.Rectangle{Width: 120, Height: 40}
		// The picker's action passes through the panes below it, which must leave it alone.
		msg := m.View().Handle(tcell.NewEventKey(tcell.KeyRune, "a", tcell.ModNone), area)
		if msg == nil {
			t.Fatal("typing produced no message")
		}
		if _, ok := msg.(paneMsg); ok {
			t.Fatalf("the picker's action was marked for a pane: %v", msg)
		}
		*m, _ = m.Update(msg)
	})
}

func TestModelUpdate(t *testing.T) {
	t.Run("erasing an edited message and submitting asks to delete it", func(t *testing.T) {
		m := newTestModel(t)
		// Only the author may delete their own message.
		if err := m.state.MyselfSet(discord.User{ID: 1}, false); err != nil {
			t.Fatal(err)
		}

		// The composer is only enabled where the user may send, which needs no permission in a direct message.
		channel := discord.Channel{ID: 1, Type: discord.DirectMessage}
		const content = "hello"
		m.messagesList.SetChannel(&channel, []discord.Message{{ID: 3, ChannelID: 1, Author: discord.User{ID: 1}, Content: content}})
		m.composer.SetChannel(&channel)
		m.setFocus(paneComposer)
		m.syncPanes()

		press(t, m, tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone))
		if !strings.Contains(composerText(t, m), "Editing") {
			t.Fatal("up did not start editing the last message")
		}

		for range content {
			press(t, m, tcell.NewEventKey(tcell.KeyBackspace2, "", tcell.ModNone))
		}

		var asked bool
		for _, msg := range press(t, m, tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)) {
			if _, ok := msg.(ui.DialogMsg); ok {
				asked = true
			}
		}

		if !asked {
			t.Fatal("submitting an erased edit did not ask to delete the message")
		}
	})

	t.Run("deleting the edited message stops the edit", func(t *testing.T) {
		m := newTestModel(t)
		message := discord.Message{ID: 3, ChannelID: 1, Author: discord.User{ID: 1}}
		m.composer.StartEdit(message)
		if !strings.Contains(composerText(t, m), "Editing") {
			t.Fatal("the composer is not showing an edit to begin with")
		}

		*m, _ = m.Update(&gateway.MessageDeleteEvent{ID: message.ID, ChannelID: message.ChannelID})
		if strings.Contains(composerText(t, m), "Editing") {
			t.Fatal("the composer kept editing a deleted message")
		}
	})

	t.Run("typing expires unless extended", func(t *testing.T) {
		m := newTestModel(t)
		t.Run("extended", func(t *testing.T) {
			m.addTyper(1)
			m.addTyper(1)
			*m, _ = m.Update(typingExpiredMsg{userID: 1})
			if _, ok := m.typers[1]; !ok {
				t.Fatal("an extended typing indicator expired early")
			}
		})
		t.Run("expired", func(t *testing.T) {
			m.typers[1] = time.Now().Add(-time.Second)
			*m, _ = m.Update(typingExpiredMsg{userID: 1})
			if _, ok := m.typers[1]; ok {
				t.Fatal("the typing indicator did not expire")
			}
		})
	})
}

func TestModelAddMessageOrNotify(t *testing.T) {
	for _, tt := range []struct {
		name          string
		focused       bool
		whenUnfocused bool
		want          bool
	}{
		{"focused", true, true, false},
		{"unfocused", false, true, true},
		{"unfocused but disabled", false, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t)
			m.cfg.Notifications.WhenUnfocused = tt.whenUnfocused
			// Loading an avatar is a command too.
			m.cfg.UI.MessagesList.Avatars.Preview = false
			m.setSelectedChannel(&discord.Channel{ID: 1})
			*m, _ = m.Update(tcell.NewEventFocus(tt.focused))
			cmd := m.addMessageOrNotify(&gateway.MessageCreateEvent{ID: 1, ChannelID: 1})
			if got := cmd != nil; got != tt.want {
				t.Fatalf("notified = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelCycleFocus(t *testing.T) {
	m := newTestModel(t)
	m.setSelectedChannel(&discord.Channel{ID: 1, GuildID: 2, Type: discord.GuildText})
	m.focused = paneGuildsTree

	// The composer is disabled without permission to send messages, so it is skipped.
	for _, test := range []struct {
		name  string
		step  pane
		panes []pane
	}{
		{"forward", 1, []pane{paneMessagesList, paneMembersTree, paneGuildsTree}},
		{"backward", -1, []pane{paneMembersTree, paneMessagesList, paneGuildsTree}},
	} {
		t.Run(test.name, func(t *testing.T) {
			m.focused = paneGuildsTree
			for i, want := range test.panes {
				t.Run(fmt.Sprintf("step %d", i+1), func(t *testing.T) {
					if m.cycleFocus(test.step); m.focused != want {
						t.Fatalf("focused = %v, want %v", m.focused, want)
					}
				})
			}
		})
	}
}

func TestModelReloadImages(t *testing.T) {
	const url = "avatar.png"
	m := newTestModel(t)
	m.kitty, m.cellWidth, m.cellHeight = true, 9, 18
	m.images[url] = tviewimage.New(image.NewAlpha(image.Rect(0, 0, 1, 1)))
	cell := m.imageCell()

	t.Run("same cell size", func(t *testing.T) {
		m.reloadImages(cell)
		if len(m.images) != 1 {
			t.Fatal("images were dropped though the cell size did not change")
		}
	})

	t.Run("new cell size", func(t *testing.T) {
		m.cellWidth, m.cellHeight = 25, 49
		m.reloadImages(cell)
		if len(m.images) != 0 {
			t.Fatal("images sized for the old cell size were kept")
		}
	})

	t.Run("stale download", func(t *testing.T) {
		m.images[url] = tviewimage.New(image.NewAlpha(image.Rect(0, 0, 1, 1)))
		// Only an accepted image is sent to kitty.
		t.Run("for the old cell size", func(t *testing.T) {
			*m, _ = m.Update(images.LoadedMsg{URL: url, Image: m.images[url], Cell: cell})
			if m.kittyImages != 0 {
				t.Fatal("an image sized for the old cell size was shown")
			}
		})
		t.Run("for the cell size", func(t *testing.T) {
			*m, _ = m.Update(images.LoadedMsg{URL: url, Image: m.images[url], Cell: m.imageCell()})
			if m.kittyImages != 1 {
				t.Fatal("an image sized for the cell size was not shown")
			}
		})
	})
}
