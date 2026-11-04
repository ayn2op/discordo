package guildstree

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/favorites"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview/tree"
)

// newTestModel returns a tree whose state knows channels, so that a bookmark resolves to one.
func newTestModel(t *testing.T, channels ...discord.Channel) Model {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}

	m := NewModel(cfg, ningen.FromState(state.New("")))
	for _, channel := range channels {
		if err := m.state.ChannelSet(&channel, false); err != nil {
			t.Fatal(err)
		}
	}

	return m
}

// bookmarked returns a store with each channel bookmarked in the slot it is given in.
func bookmarked(channelIDs ...discord.ChannelID) favorites.Store {
	store := favorites.Store{}
	for slot, channelID := range channelIDs {
		store = store.With(slot, favorites.Channel{ChannelID: channelID})
	}

	return store
}

// names returns the text of each child of node.
func names(node *tree.Node) []string {
	children := node.Children()
	texts := make([]string, 0, len(children))
	for _, child := range children {
		var text strings.Builder
		for _, span := range child.Spans() {
			text.WriteString(span.Text)
		}
		texts = append(texts, text.String())
	}

	return texts
}

func TestModelSetFavorites(t *testing.T) {
	general := discord.Channel{ID: 1, Name: "general", Type: discord.GuildText, GuildID: 7}
	dev := discord.Channel{ID: 2, Name: "dev", Type: discord.GuildText, GuildID: 7}

	t.Run("lists the bookmarked channels in slot order", func(t *testing.T) {
		m := newTestModel(t, general, dev)
		m.SetFavorites(bookmarked(dev.ID, general.ID))

		if got, want := names(m.favoritesRootNode), []string{"f1 #dev", "f2 #general"}; !equal(got, want) {
			t.Fatalf("favorites = %q, want %q", got, want)
		}
	})

	t.Run("shows the section only once something is bookmarked", func(t *testing.T) {
		m := newTestModel(t, general)
		if shown(m) {
			t.Fatal("the section is in the tree with nothing bookmarked")
		}

		m.SetFavorites(bookmarked(general.ID))
		if !shown(m) {
			t.Fatal("the section is not in the tree with a channel bookmarked")
		}

		m.SetFavorites(favorites.Store{})
		if shown(m) {
			t.Fatal("the section stayed in the tree with nothing bookmarked")
		}
	})

	t.Run("the section comes first", func(t *testing.T) {
		m := newTestModel(t, general)
		m.reset()
		m.SetFavorites(bookmarked(general.ID))

		if first := m.root.Children()[0]; first != m.favoritesRootNode {
			t.Fatalf("the first section is %v, want the favorites", names(first))
		}
	})

	t.Run("an empty slot takes no row", func(t *testing.T) {
		m := newTestModel(t, general)
		m.SetFavorites(favorites.Store{}.With(2, favorites.Channel{ChannelID: general.ID}))

		if got, want := names(m.favoritesRootNode), []string{"f3 #general"}; !equal(got, want) {
			t.Fatalf("favorites = %q, want %q", got, want)
		}
	})

	t.Run("a channel the state does not know takes no row", func(t *testing.T) {
		m := newTestModel(t, general)
		m.SetFavorites(bookmarked(general.ID, 404))

		if got, want := names(m.favoritesRootNode), []string{"f1 #general"}; !equal(got, want) {
			t.Fatalf("favorites = %q, want %q", got, want)
		}
	})

	t.Run("a channel that opens no messages takes no row", func(t *testing.T) {
		forum := discord.Channel{ID: 3, Name: "help", Type: discord.GuildForum, GuildID: 7}
		m := newTestModel(t, general, forum)
		m.SetFavorites(bookmarked(forum.ID, general.ID))

		if got, want := names(m.favoritesRootNode), []string{"f2 #general"}; !equal(got, want) {
			t.Fatalf("favorites = %q, want %q", got, want)
		}
	})

	t.Run("a bookmarked channel keeps the node indexed under its guild", func(t *testing.T) {
		m := newTestModel(t, general)
		// The node under the guild, as the tree indexes it once the guild is expanded.
		own := tree.NewNode("").SetReference(general.ID)
		m.channelNodes[general.ID] = own

		m.SetFavorites(bookmarked(general.ID))
		if m.channelNodes[general.ID] != own {
			t.Fatal("the bookmark took the channel's place in the index")
		}
		if m.favoriteNodes[general.ID] == own {
			t.Fatal("the bookmark and the channel share a node")
		}
		if got := len(m.nodesShowing(general.ID)); got != 2 {
			t.Fatalf("the channel is shown by %d nodes, want 2", got)
		}
	})
}

func TestModelCurrentChannelID(t *testing.T) {
	general := discord.Channel{ID: 1, Name: "general", Type: discord.GuildText, GuildID: 7}
	for _, test := range []struct {
		name    string
		current *tree.Node
		want    discord.ChannelID
		wantOk  bool
	}{
		{"nothing highlighted", nil, discord.NullChannelID, false},
		{"a channel", tree.NewNode("").SetReference(general.ID), general.ID, true},
		{"a bookmark", tree.NewNode("").SetReference(favoriteNode{slot: 0, channelID: general.ID}), general.ID, true},
		{"a guild", tree.NewNode("").SetReference(general.GuildID), discord.NullChannelID, false},
		{"the favorites section", tree.NewNode("").SetReference(favoritesRoot{}), discord.NullChannelID, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			m := newTestModel(t, general)
			m.current = test.current

			got, ok := m.CurrentChannelID()
			if ok != test.wantOk {
				t.Fatalf("ok = %v, want %v", ok, test.wantOk)
			}
			if got != test.want {
				t.Fatalf("got = %v, want %v", got, test.want)
			}
		})
	}
}

// shown reports whether the favorites section is among the tree's sections.
func shown(m Model) bool {
	for _, child := range m.root.Children() {
		if child == m.favoritesRootNode {
			return true
		}
	}

	return false
}

func equal(got, want []string) bool {
	if len(got) != len(want) {
		return false
	}
	for index := range got {
		if got[index] != want[index] {
			return false
		}
	}

	return true
}
