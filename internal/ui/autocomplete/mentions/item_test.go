package mentions

import (
	"image"
	"path/filepath"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui/autocomplete"
	"github.com/ayn2op/ningen/v3"
	tviewimage "github.com/ayn2op/tview/image"
)

func TestItems(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}

	s := ningen.FromState(state.NewWithStore("", defaultstore.New()))
	if err := s.MyselfSet(discord.User{ID: 1, Username: "me"}, false); err != nil {
		t.Fatal(err)
	}

	channel := discord.Channel{ID: 1, Type: discord.DirectMessage, DMRecipients: []discord.User{{ID: 2, Username: "ann"}, {ID: 3, Username: "bob"}}}
	for id, author := range []discord.UserID{2, 3, 2, 1} {
		message := discord.Message{ID: discord.MessageID(id + 1), ChannelID: channel.ID, Author: discord.User{ID: author}}
		if err := s.MessageSet(&message, false); err != nil {
			t.Fatal(err)
		}
	}

	t.Run("recent authors", func(t *testing.T) {
		if got := len(loadItems(t, cfg, s, &channel, "", 10)); got != 2 {
			t.Errorf("got %d authors, want the 2 others", got)
		}
	})

	t.Run("limit", func(t *testing.T) {
		if got := len(loadItems(t, cfg, s, &channel, "", 1)); got != 1 {
			t.Errorf("got %d authors, want 1", got)
		}
	})

	t.Run("recipients", func(t *testing.T) {
		items := loadItems(t, cfg, s, &channel, "bob", 10)
		if len(items) != 1 || items[0].Insert() != "@bob" {
			t.Errorf("got %v, want @bob", items)
		}
	})
}

// loadItems calls Items and checks it loads avatars only if they are shown.
func loadItems(t *testing.T, cfg *config.Config, s *ningen.State, channel *discord.Channel, query string, limit int) []autocomplete.Item {
	t.Helper()
	items, loadImages := Items(cfg, s, channel, query, limit)
	if shown := len(items) > 0 && cfg.UI.MessagesList.Avatars.Preview; shown != (loadImages != nil) {
		t.Errorf("loads avatars = %t, want %t", loadImages != nil, shown)
	}
	return items
}

func TestMentionLine(t *testing.T) {
	loaded := mention{name: "ann", avatar: "ann.png"}
	// Only the avatar of loaded is loaded.
	images := map[discord.URL]tviewimage.Widget{loaded.avatar: tviewimage.New(image.NewAlpha(image.Rect(0, 0, 2, 2))).Width(2)}
	for _, test := range []struct {
		name    string
		mention mention
		want    int
	}{
		{"loaded", loaded, 4},
		{"not loaded", mention{name: "bob", avatar: "bob.png"}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.mention.Line(images)
			if len(got) != test.want || got[len(got)-1].Text != test.mention.name {
				t.Errorf("Line = %v, want %d spans ending with the name", got, test.want)
			}
		})
	}
}
