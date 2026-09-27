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
	return NewModel(cfg, "")
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
		m.Update(msg)
	})
}

func TestModelUpdate(t *testing.T) {
	t.Run("typing expires unless extended", func(t *testing.T) {
		m := newTestModel(t)
		m.addTyper(1)
		m.addTyper(1)
		m.Update(typingExpiredMsg{userID: 1})
		if _, ok := m.typers[1]; !ok {
			t.Fatal("an extended typing indicator expired early")
		}
		m.typers[1] = time.Now().Add(-time.Second)
		m.Update(typingExpiredMsg{userID: 1})
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
			m.Update(tcell.NewEventFocus(tt.focused))
			cmd := m.addMessageOrNotify(&gateway.MessageCreateEvent{Message: discord.Message{ID: 1, ChannelID: 1}})
			if got := cmd != nil; got != tt.want {
				t.Fatalf("notified = %v, want %v", got, tt.want)
			}
		})
	}
}
