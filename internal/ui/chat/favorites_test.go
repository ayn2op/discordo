package chat

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/favorites"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/vt"
)

// favoritesFile points the model's bookmarks at a file of the test's own and returns its path.
func favoritesFile(t *testing.T, m *Model) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "favorites.toml")
	m.favorites = favorites.Open(path)
	return path
}

func TestFavoriteSlot(t *testing.T) {
	keybinds := config.Default().Keybinds
	for _, test := range []struct {
		name     string
		key      tview.KeyMsg
		keybinds []config.Keybind
		want     int
	}{
		{"the first digit takes the first slot", tcell.NewEventKey(tcell.KeyRune, "1", tcell.ModAlt), keybinds.Favorites, 0},
		{"zero takes the last slot", tcell.NewEventKey(tcell.KeyRune, "0", tcell.ModAlt), keybinds.Favorites, 9},
		{"a function key reaches the slot of its number", tcell.NewEventKey(tcell.KeyF3, "", tcell.ModNone), keybinds.Favorites, 2},
		{"a digit with ctrl reaches the slot of its digit", tcell.NewEventKey(tcell.KeyRune, "3", tcell.ModCtrl), keybinds.Favorites, 2},
		// A terminal without an enhanced keyboard protocol sends a control code for ctrl with a digit,
		// which is a key of its own and so reaches no slot.
		{"the escape ctrl and 3 send instead reaches no slot", tcell.NewEventKey(tcell.KeyEscape, "", tcell.ModNone), keybinds.Favorites, -1},
		{"the last function key reaches the last slot", tcell.NewEventKey(tcell.KeyF10, "", tcell.ModNone), keybinds.Favorites, 9},
		{"setting has its own keys", tcell.NewEventKey(tcell.KeyRune, "!", tcell.ModAlt), keybinds.SetFavorites, 0},
		{"opening does not answer a setting key", tcell.NewEventKey(tcell.KeyRune, "!", tcell.ModAlt), keybinds.Favorites, -1},
		{"setting does not answer an opening key", tcell.NewEventKey(tcell.KeyRune, "1", tcell.ModAlt), keybinds.SetFavorites, -1},
		{"an unbound key takes no slot", tcell.NewEventKey(tcell.KeyRune, "1", tcell.ModNone), keybinds.Favorites, -1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := favoriteSlot(test.key, test.keybinds); got != test.want {
				t.Fatalf("got = %d, want = %d", got, test.want)
			}
		})
	}
}

func TestModelSetFavorite(t *testing.T) {
	const slot = 0
	channel := discord.Channel{ID: 9, GuildID: 7}

	t.Run("bookmarks the open channel", func(t *testing.T) {
		m := newTestModel(t)
		path := favoritesFile(t, m)
		m.setSelectedChannel(&channel)

		cmd := m.setFavorite(slot)
		if got := m.favorites.Channel(slot); got != channel.ID {
			t.Fatalf("slot %d = %v, want %v", slot, got, channel.ID)
		}

		cmd()
		if got := favorites.Open(path).Channel(slot); got != channel.ID {
			t.Fatalf("saved slot %d = %v, want %v", slot, got, channel.ID)
		}
	})

	t.Run("empties the slot the open channel is bookmarked in", func(t *testing.T) {
		m := newTestModel(t)
		path := favoritesFile(t, m)
		m.setSelectedChannel(&channel)

		m.setFavorite(slot)
		cmd := m.setFavorite(slot)
		if got := m.favorites.Channel(slot); got.IsValid() {
			t.Fatalf("slot %d = %v, want an empty slot", slot, got)
		}

		cmd()
		if got := favorites.Open(path).Channel(slot); got.IsValid() {
			t.Fatalf("saved slot %d = %v, want an empty slot", slot, got)
		}
	})

	t.Run("bookmarks another channel in a slot that is taken", func(t *testing.T) {
		m := newTestModel(t)
		favoritesFile(t, m)
		m.setSelectedChannel(&channel)
		m.setFavorite(slot)

		other := discord.Channel{ID: 11, GuildID: 7}
		m.setSelectedChannel(&other)
		m.setFavorite(slot)
		if got := m.favorites.Channel(slot); got != other.ID {
			t.Fatalf("slot %d = %v, want %v", slot, got, other.ID)
		}
	})

	t.Run("bookmarks nothing without an open channel", func(t *testing.T) {
		m := newTestModel(t)
		favoritesFile(t, m)

		if cmd := m.setFavorite(slot); cmd != nil {
			t.Fatal("bookmarking with no channel open was saved")
		}
		if got := m.favorites.Channel(slot); got.IsValid() {
			t.Fatalf("slot %d = %v, want an empty slot", slot, got)
		}
	})
}

// guildsTreeSize is the area the guilds tree is drawn in to read what it shows.
var guildsTreeSize = tview.Rectangle{Width: 40, Height: 10}

// guildsTreeText returns everything the guilds tree draws.
func guildsTreeText(t *testing.T, m *Model) string {
	t.Helper()
	screen, err := tcell.NewTerminfoScreenFromTty(vt.NewMockTerm(vt.MockOptSize{X: vt.Col(guildsTreeSize.Width), Y: vt.Row(guildsTreeSize.Height)}))
	if err != nil {
		t.Fatal(err)
	}
	if err := screen.Init(); err != nil {
		t.Fatal(err)
	}
	defer screen.Fini()

	m.guildsTree.View().Draw(screen, guildsTreeSize)

	var drawn strings.Builder
	for y := range guildsTreeSize.Height {
		for x := range guildsTreeSize.Width {
			str, _, _ := screen.Get(x, y)
			drawn.WriteString(str)
		}
	}

	return drawn.String()
}

func TestModelSyncPanes(t *testing.T) {
	t.Run("a bookmarked channel is listed in the guilds tree", func(t *testing.T) {
		m := newTestModel(t)
		favoritesFile(t, m)
		channel := discord.Channel{ID: 9, GuildID: 7, Name: "general", Type: discord.GuildText}
		if err := m.state.ChannelSet(&channel, false); err != nil {
			t.Fatal(err)
		}

		m.setSelectedChannel(&channel)
		if drawn := guildsTreeText(t, m); strings.Contains(drawn, "Favorites") {
			t.Fatalf("the favorites section is shown with nothing bookmarked: %q", drawn)
		}

		m.setFavorite(0)
		*m, _ = m.Update(nil)

		drawn := guildsTreeText(t, m)
		for _, want := range []string{"Favorites", "f1 #general"} {
			if !strings.Contains(drawn, want) {
				t.Fatalf("the guilds tree shows %q, want it to hold %q", drawn, want)
			}
		}
	})
}

func TestModelFavorite(t *testing.T) {
	t.Run("a setting key bookmarks the open channel", func(t *testing.T) {
		m := newTestModel(t)
		favoritesFile(t, m)
		channel := discord.Channel{ID: 9, GuildID: 7}
		m.setSelectedChannel(&channel)

		*m, _ = m.Update(tcell.NewEventKey(tcell.KeyRune, "@", tcell.ModAlt))
		if got := m.favorites.Channel(1); got != channel.ID {
			t.Fatalf("slot 1 = %v, want %v", got, channel.ID)
		}
	})

	t.Run("an empty slot opens nothing", func(t *testing.T) {
		m := newTestModel(t)
		favoritesFile(t, m)

		cmd, ok := m.favorite(tcell.NewEventKey(tcell.KeyRune, "1", tcell.ModAlt))
		if !ok {
			t.Fatal("the key of an empty slot was left to the panes")
		}
		if cmd != nil {
			t.Fatal("an empty slot was opened")
		}
	})

	t.Run("a key of no slot is left to the panes", func(t *testing.T) {
		m := newTestModel(t)
		if _, ok := m.favorite(tcell.NewEventKey(tcell.KeyRune, "1", tcell.ModNone)); ok {
			t.Fatal("a key bound to no slot was taken as a favorite")
		}
	})
}
