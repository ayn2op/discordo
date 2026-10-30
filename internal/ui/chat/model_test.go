package chat

import (
	"errors"
	"path/filepath"
	"testing"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui/chat/messageslist"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
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

// maxCommandHops bounds drain, so a loop of commands fails the test rather than hanging it.
const maxCommandHops = 16

// drain applies msg and whatever the commands it returns produce, as the event loop does.
func drain(t *testing.T, m *Model, msg tview.Msg) {
	t.Helper()
	for range maxCommandHops {
		if msg == nil {
			return
		}

		var cmd tview.Cmd
		*m, cmd = m.Update(msg)
		if cmd == nil {
			return
		}

		msg = cmd()
	}
	t.Fatal("the commands did not settle")
}

func TestModelHandle(t *testing.T) {
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
	t.Run("a reminder time that cannot be read keeps its prompt open", func(t *testing.T) {
		m := newTestModel(t)
		drain(t, m, messageslist.ShowReminderPromptMsg{
			Set: func(string) (tview.Cmd, error) { return nil, errors.New("unreadable") },
		})
		if m.overlay != overlayReminderPrompt {
			t.Fatal("the prompt did not open")
		}

		// Typing a time it cannot read and submitting must leave the prompt up to be corrected.
		area := tview.Rectangle{Width: 120, Height: 40}
		for _, event := range []tview.Msg{
			tcell.NewEventKey(tcell.KeyRune, "x", tcell.ModNone),
			tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone),
		} {
			drain(t, m, m.View().Handle(event, area))
		}

		if m.overlay != overlayReminderPrompt {
			t.Fatal("the prompt closed on a time it could not read")
		}
	})

	t.Run("typing expires unless extended", func(t *testing.T) {
		m := newTestModel(t)
		m.addTyper(1)
		m.addTyper(1)
		*m, _ = m.Update(typingExpiredMsg{userID: 1})
		if _, ok := m.typers[1]; !ok {
			t.Fatal("an extended typing indicator expired early")
		}
		m.typers[1] = time.Now().Add(-time.Second)
		*m, _ = m.Update(typingExpiredMsg{userID: 1})
		if _, ok := m.typers[1]; ok {
			t.Fatal("the typing indicator did not expire")
		}
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
	for _, want := range []pane{paneMessagesList, paneMembersTree, paneGuildsTree} {
		if m.cycleFocus(1); m.focused != want {
			t.Fatalf("focused = %v, want %v", m.focused, want)
		}
	}
	for _, want := range []pane{paneMembersTree, paneMessagesList, paneGuildsTree} {
		if m.cycleFocus(-1); m.focused != want {
			t.Fatalf("focused = %v, want %v", m.focused, want)
		}
	}
}
