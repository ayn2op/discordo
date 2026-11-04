package favorites

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/consts"
)

// write puts contents in a favorites file of its own and returns its path.
func write(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), fileName)
	if err := os.WriteFile(path, []byte(contents), os.ModePerm); err != nil {
		t.Fatal(err)
	}

	return path
}

func TestDefaultPath(t *testing.T) {
	t.Run("sits next to the config", func(t *testing.T) {
		t.Setenv("AppData", "")
		t.Setenv("HOME", "")
		t.Setenv("home", "")
		t.Setenv("XDG_CONFIG_HOME", "")

		// filepath.Join strips the leading dot.
		got := DefaultPath()
		want := filepath.Join(".", consts.Name, fileName)
		if got != want {
			t.Fatalf("got = %v, want = %v", got, want)
		}
	})
}

func TestOpen(t *testing.T) {
	t.Run("reads the bookmarked ids", func(t *testing.T) {
		path := write(t, "[[slots]]\nguild_id = \"7\"\nchannel_id = \"9\"\n")
		store := Open(path)
		if got, want := store.Channel(0), discord.ChannelID(9); got != want {
			t.Fatalf("slot 0 = %v, want %v", got, want)
		}
	})

	t.Run("keeps the slots in order", func(t *testing.T) {
		path := write(t, "[[slots]]\n\n[[slots]]\nchannel_id = \"9\"\n")
		store := Open(path)
		if got := store.Channel(0); got.IsValid() {
			t.Fatalf("slot 0 = %v, want an empty slot", got)
		}
		if got, want := store.Channel(1), discord.ChannelID(9); got != want {
			t.Fatalf("slot 1 = %v, want %v", got, want)
		}
	})

	t.Run("leaves a slot whose id is not a snowflake empty", func(t *testing.T) {
		path := write(t, "[[slots]]\nchannel_id = \"general\"\n")
		if got := Open(path).Channel(0); got.IsValid() {
			t.Fatalf("slot 0 = %v, want an empty slot", got)
		}
	})

	t.Run("starts empty when the file does not exist", func(t *testing.T) {
		store := Open(filepath.Join(t.TempDir(), fileName))
		if got := store.Channel(0); got.IsValid() {
			t.Fatalf("slot 0 = %v, want an empty slot", got)
		}
	})

	t.Run("starts empty when the file is invalid", func(t *testing.T) {
		store := Open(write(t, "slots ="))
		if got := store.Channel(0); got.IsValid() {
			t.Fatalf("slot 0 = %v, want an empty slot", got)
		}
	})
}

func TestStoreChannel(t *testing.T) {
	store := Store{channels: []Channel{{ChannelID: 9}}}
	for _, test := range []struct {
		name string
		slot int
	}{
		{"before the first slot", -1},
		{"past the last slot", 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := store.Channel(test.slot); got.IsValid() {
				t.Fatalf("slot %d = %v, want an empty slot", test.slot, got)
			}
		})
	}
}

func TestStoreWith(t *testing.T) {
	t.Run("bookmarks in a slot", func(t *testing.T) {
		store := Store{}.With(0, Channel{GuildID: 7, ChannelID: 9})
		if got, want := store.Channel(0), discord.ChannelID(9); got != want {
			t.Fatalf("slot 0 = %v, want %v", got, want)
		}
	})

	t.Run("empties a slot", func(t *testing.T) {
		store := Store{}.With(0, Channel{ChannelID: 9}).With(0, Channel{})
		if got := store.Channel(0); got.IsValid() {
			t.Fatalf("slot 0 = %v, want an empty slot", got)
		}
	})

	t.Run("leaves the slots up to a later one empty", func(t *testing.T) {
		store := Store{}.With(2, Channel{ChannelID: 9})
		for slot := range 2 {
			if got := store.Channel(slot); got.IsValid() {
				t.Fatalf("slot %d = %v, want an empty slot", slot, got)
			}
		}
		if got, want := store.Channel(2), discord.ChannelID(9); got != want {
			t.Fatalf("slot 2 = %v, want %v", got, want)
		}
	})

	t.Run("ignores a slot before the first", func(t *testing.T) {
		if store := (Store{}).With(-1, Channel{ChannelID: 9}); len(store.channels) != 0 {
			t.Fatalf("channels = %v, want none", store.channels)
		}
	})

	t.Run("leaves the store it is called on alone", func(t *testing.T) {
		store := Store{}.With(0, Channel{ChannelID: 9})
		store.With(0, Channel{ChannelID: 11})
		if got, want := store.Channel(0), discord.ChannelID(9); got != want {
			t.Fatalf("slot 0 = %v, want %v", got, want)
		}
	})
}

func TestStoreSave(t *testing.T) {
	t.Run("round trips through the file", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), fileName)
		store := Open(path).With(0, Channel{GuildID: 7, ChannelID: 9}).With(2, Channel{ChannelID: 11})
		if err := store.Save(); err != nil {
			t.Fatal(err)
		}

		reopened := Open(path)
		for slot, want := range map[int]discord.ChannelID{0: 9, 2: 11} {
			if got := reopened.Channel(slot); got != want {
				t.Fatalf("slot %d = %v, want %v", slot, got, want)
			}
		}
		if got := reopened.Channel(1); got.IsValid() {
			t.Fatalf("slot 1 = %v, want an empty slot", got)
		}
	})

	t.Run("creates the directory", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), consts.Name, fileName)
		if err := Open(path).With(0, Channel{ChannelID: 9}).Save(); err != nil {
			t.Fatal(err)
		}

		if got, want := Open(path).Channel(0), discord.ChannelID(9); got != want {
			t.Fatalf("slot 0 = %v, want %v", got, want)
		}
	})

	t.Run("returns the error of a path it cannot write", func(t *testing.T) {
		if err := (Store{path: "bad\x00path"}).Save(); err == nil {
			t.Fatal("expected error")
		}
	})
}
