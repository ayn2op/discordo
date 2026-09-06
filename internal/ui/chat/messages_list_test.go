package chat

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
)

func TestMessagesListBuildItem(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg, "")
	m.SetSelectedChannel(&discord.Channel{ID: 1})
	ml := m.messagesList
	ml.setMessages([]discord.Message{{ID: 3}, {ID: 2}})
	if ml.items[ml.messageIndex(-1, 1)].message.ID != 2 {
		t.Fatal("messages are not oldest first")
	}
	view := ml.buildItem(ml.messageIndex(len(ml.items), -1))
	if ml.buildItem(ml.messageIndex(len(ml.items), -1)) != view {
		t.Fatal("message view was not reused")
	}
	ml.selectBottom()
	ml.Update(olderMessagesLoadedMsg{ChannelID: 1, Older: []discord.Message{{ID: 1}}})
	selected, ok := ml.selectedMessage()
	if !ok || selected.ID != 3 || ml.buildItem(ml.messageIndex(len(ml.items), -1)) != view {
		t.Fatal("prepend lost selection or cached view")
	}
	ml.deleteMessage(ml.messageIndex(-1, 1))
	if ml.buildItem(ml.messageIndex(len(ml.items), -1)) != view {
		t.Fatal("deletion discarded another message's view")
	}
	ml.setMessage(ml.messageIndex(len(ml.items), -1), discord.Message{ID: 3, Content: "edited"})
	edited := ml.buildItem(ml.messageIndex(len(ml.items), -1))
	if edited == view {
		t.Fatal("edit reused a stale view")
	}
	ml.invalidateRenderedMessages()
	if ml.buildItem(ml.messageIndex(len(ml.items), -1)) == edited {
		t.Fatal("global invalidation reused a stale view")
	}
	ml.reset()
	if len(ml.items) != 0 {
		t.Fatal("reset retained messages or rows")
	}
}

func TestMessagesListRebuildItems(t *testing.T) {
	for _, tt := range []struct {
		name       string
		separators bool
	}{
		{name: "separators_disabled", separators: false},
		{name: "separators_enabled", separators: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
			if err != nil {
				t.Fatal(err)
			}
			cfg.DateSeparator.Enabled = tt.separators
			m := NewModel(cfg, "")
			m.SetSelectedChannel(&discord.Channel{ID: 1})
			ml := m.messagesList
			var messages []discord.Message
			if err := json.Unmarshal([]byte(`[
   {"id":"3","timestamp":"2026-09-03T12:00:00Z"},
   {"id":"2","timestamp":"2026-09-02T12:00:00Z"},
   {"id":"1","timestamp":"2026-09-01T12:00:00Z"}
  ]`), &messages); err != nil {
				t.Fatal(err)
			}
			ml.setMessages(messages[:2])
			assertItems := func(want ...discord.MessageID) {
				t.Helper()
				var got []discord.MessageID
				for i, item := range ml.items {
					if item.separator {
						if !tt.separators || i+1 == len(ml.items) || ml.items[i+1].separator || item.timestamp != ml.items[i+1].message.Timestamp {
							t.Fatalf("item %d: orphaned or incorrect separator", i)
						}
					} else {
						got = append(got, item.message.ID)
					}
				}
				if !slices.Equal(got, want) {
					t.Fatalf("messages = %v, want %v", got, want)
				}
				expected := len(want)
				if tt.separators {
					expected *= 2
				}
				if len(ml.items) != expected {
					t.Fatalf("item count = %d, want %d", len(ml.items), expected)
				}
			}
			assertSelected := func(want discord.MessageID) {
				t.Helper()
				message, ok := ml.selectedMessage()
				if !ok || message.ID != want {
					t.Fatalf("selected = %v, want %v", message, want)
				}
			}
			assertItems(2, 3)
			ml.selectTop()
			assertSelected(2)
			ml.selectDown()
			assertSelected(3)
			ml.selectUp()
			assertSelected(2)
			ml.Update(olderMessagesLoadedMsg{ChannelID: 1, Older: messages[2:]})
			assertItems(1, 2, 3)
			assertSelected(1)
			ml.selectDown()
			ml.deleteMessage(ml.Cursor())
			assertItems(1, 3)
			assertSelected(1)
			ml.deleteMessage(ml.Cursor())
			assertItems(3)
			assertSelected(3)
			ml.deleteMessage(ml.Cursor())
			assertItems()
			if ml.Cursor() != -1 {
				t.Fatal("empty list retained selection")
			}
		})
	}
}
