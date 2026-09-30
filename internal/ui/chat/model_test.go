package chat

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/config"
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
			m.setSelectedChannel(&discord.Channel{ID: 1})
			*m, _ = m.Update(tcell.NewEventFocus(tt.focused))
			cmd := m.addMessageOrNotify(&gateway.MessageCreateEvent{Message: discord.Message{ID: 1, ChannelID: 1}})
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
