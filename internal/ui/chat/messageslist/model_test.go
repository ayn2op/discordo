package messageslist

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/layout"
)

func testConfig(t *testing.T) *config.Config {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

// newTestModel returns a list showing channel 1, logged in as user 1.
func newTestModel(t *testing.T, cfg *config.Config) *Model {
	t.Helper()
	ml := NewModel(cfg, ningen.FromState(state.New("")))
	if err := ml.state.Cabinet.MyselfSet(discord.User{ID: 1}, false); err != nil {
		t.Fatal(err)
	}
	ml.channel = &discord.Channel{ID: 1}
	return &ml
}

func TestModelBuildItem(t *testing.T) {
	ml := newTestModel(t, testConfig(t))
	ml.setMessages([]discord.Message{{ID: 3}, {ID: 2}})
	if ml.items[ml.messageIndex(-1, 1)].message.ID != 2 {
		t.Fatal("messages are not oldest first")
	}
	// text returns the wrapped text kept for the newest message once it is shown.
	text := func() *messageView {
		index := ml.messageIndex(len(ml.items), -1)
		ml.buildItem(index)
		return ml.items[index].text
	}
	view := text()
	if text() != view {
		t.Fatal("message text was not reused")
	}
	if ml.selectTop() == nil || ml.selectTop() != nil || ml.selectUp() != nil {
		t.Fatal("older messages were not fetched once on reaching the top")
	}
	ml.selectBottom()
	if _, cmd := ml.Update(listMsg(0)); cmd == nil {
		t.Fatal("clicking the top message did not fetch older messages")
	}
	ml.selectBottom()
	*ml, _ = ml.Update(olderMessagesLoadedMsg{ChannelID: 1, Older: []discord.Message{{ID: 1}}})
	selected, ok := ml.selectedMessage()
	if !ok || selected.ID != 3 || text() != view {
		t.Fatal("prepend lost selection or cached text")
	}
	ml.deleteMessage(ml.messageIndex(-1, 1))
	if text() != view {
		t.Fatal("deletion discarded another message's text")
	}
	ml.setMessage(ml.messageIndex(len(ml.items), -1), discord.Message{ID: 3, Content: "edited"})
	edited := text()
	if edited == view {
		t.Fatal("edit reused stale text")
	}
	ml.InvalidateRendered()
	if text() == edited {
		t.Fatal("global invalidation reused stale text")
	}
	ml.reset()
	if len(ml.items) != 0 {
		t.Fatal("reset retained messages or rows")
	}
}

func TestModelRebuildItems(t *testing.T) {
	for _, tt := range []struct {
		name       string
		separators bool
	}{
		{name: "separators_disabled", separators: false},
		{name: "separators_enabled", separators: true},
	} {
		t.Run(tt.name, func(t *testing.T) {
			cfg := testConfig(t)
			cfg.DateSeparator.Enabled = tt.separators
			ml := newTestModel(t, cfg)
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
						if !tt.separators || i+1 == len(ml.items) || ml.items[i+1].separator {
							t.Fatalf("item %d: orphaned separator", i)
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
			*ml, _ = ml.Update(olderMessagesLoadedMsg{ChannelID: 1, Older: messages[2:]})
			assertItems(1, 2, 3)
			assertSelected(2)
			ml.deleteMessage(ml.cursor)
			assertItems(1, 3)
			assertSelected(1)
			ml.deleteMessage(ml.cursor)
			assertItems(3)
			assertSelected(3)
			ml.deleteMessage(ml.cursor)
			assertItems()
			if ml.cursor != -1 {
				t.Fatal("empty list retained selection")
			}
		})
	}
}

func TestModelSelectLastOwn(t *testing.T) {
	messages := []discord.Message{
		{ID: 1, ChannelID: 1, Author: discord.User{ID: 1}, Content: "old message"},
		{ID: 2, ChannelID: 1, Author: discord.User{ID: 1}, Content: "latest message"},
		{ID: 3, ChannelID: 1, Author: discord.User{ID: 2}, Content: "someone else"},
		{ID: 4, ChannelID: 1, Author: discord.User{ID: 1}, Type: discord.ChannelPinnedMessage, Content: "system"},
		{ID: 5, ChannelID: 1, Author: discord.User{ID: 1}},
	}
	for _, tt := range []struct {
		name     string
		messages []discord.Message
		want     string
	}{
		{"latest", messages, "latest message"},
		{"old", messages[:1], "old message"},
		{"no own message", messages[2:3], ""},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ml := newTestModel(t, testConfig(t))
			newest := slices.Clone(tt.messages)
			slices.Reverse(newest)
			ml.setMessages(newest)
			message, _ := ml.SelectLastOwn()
			selected, _ := ml.selectedMessage()
			if message.Content != tt.want || tt.want != "" && selected.Content != tt.want {
				t.Fatalf("returned %q, selected %v, want %q", message.Content, selected, tt.want)
			}
		})
	}
}

func TestWaveformRail(t *testing.T) {
	waveform := make([]byte, 256)
	waveform[255] = 255

	if got, want := strings.Join(waveformRail(waveform), ""), strings.Repeat("▁", 28)+"█"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestMessageURLs(t *testing.T) {
	message := discord.Message{Content: "[a](file:///etc/passwd) [b](ms-msdt:id) [c](HTTPS://example.com) https://example.org"}

	if got, want := messageURLs(message), []string{"HTTPS://example.com", "https://example.org"}; !slices.Equal(got, want) {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestModelDrawLayout(t *testing.T) {
	cfg := testConfig(t)
	// A layout without a reference time is drawn as it is, whatever the time zone.
	cfg.Timestamps.Format = "T"
	ml := newTestModel(t, cfg)
	message := discord.Message{Author: discord.User{Username: "a"}, Content: "hi", EditedTimestamp: discord.NowTimestamp()}

	for compact, want := range map[bool]string{false: "a T (edited)\nhi\n", true: "T a hi (edited)\n"} {
		cfg.UI.MessagesList.Compact = compact

		var got strings.Builder
		for _, line := range ml.renderMessage(message, tview.Style{}, 80) {
			for _, segment := range line {
				got.WriteString(segment.Text)
			}

			got.WriteByte('\n')
		}

		if got.String() != want {
			t.Fatalf("got = %q, want = %q", got.String(), want)
		}
	}
}

func TestMessageViewLayout(t *testing.T) {
	cfg := testConfig(t)
	cfg.UI.MessagesList.Compact = true
	ml := newTestModel(t, cfg)
	limits := layout.Limits{Max: layout.Size{Width: 80}, Infinite: layout.Axes{Height: true}}

	// The avatar is 2 rows, below the row of a reply.
	for want, messageType := range map[int]discord.MessageType{2: discord.DefaultMessage, 3: discord.InlinedReplyMessage} {
		ml.SetChannel(ml.channel, []discord.Message{{ID: 1, Type: messageType, Content: "hi"}})
		if got := ml.buildItem(ml.messageIndex(-1, 1)).Layout(limits).Height; got != want {
			t.Fatalf("height = %d, want %d", got, want)
		}
	}
}
