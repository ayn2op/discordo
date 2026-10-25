package emojis

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
)

func TestItems(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}

	s := ningen.FromState(state.New(""))

	t.Run("aliases once", func(t *testing.T) {
		// 👍 is named thumbsup, thumbup, +1 and more.
		count := 0
		items, _ := Items(cfg, s, discord.NullGuildID, "thumbsup", 100)
		for _, item := range items {
			if item.Insert() == "👍" {
				count++
			}
		}

		if count != 1 {
			t.Errorf("👍 shown %d times, want 1", count)
		}
	})

	t.Run("limit", func(t *testing.T) {
		if items, _ := Items(cfg, s, discord.NullGuildID, "face", 3); len(items) != 3 {
			t.Errorf("got %d items, want 3", len(items))
		}

		if items, _ := Items(cfg, s, discord.NullGuildID, "face", 0); len(items) != 0 {
			t.Errorf("got %d items, want 0", len(items))
		}
	})

	t.Run("nothing typed", func(t *testing.T) {
		items, _ := Items(cfg, s, discord.NullGuildID, "", 3)
		var got []string
		for _, item := range items {
			got = append(got, item.Insert())
		}

		if want := []string{"😀", "😃", "😄"}; !slices.Equal(got, want) {
			t.Errorf("got %v, want %v", got, want)
		}
	})

	t.Run("best first", func(t *testing.T) {
		items, _ := Items(cfg, s, discord.NullGuildID, "joy", 1)
		if len(items) != 1 || items[0].Insert() != "😂" {
			t.Errorf("got %v, want 😂", items)
		}
	})
}
