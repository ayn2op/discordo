package emojis

import (
	"image"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	tviewimage "github.com/ayn2op/tview/image"
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

func TestCustomEmojiLine(t *testing.T) {
	loaded := customEmoji{emoji: discord.Emoji{ID: 1}, name: "cat"}
	// Only the image of loaded is loaded.
	images := map[discord.URL]tviewimage.Widget{loaded.image(): tviewimage.New(image.NewAlpha(image.Rect(0, 0, 2, 2))).Width(2)}
	for _, test := range []struct {
		name  string
		emoji customEmoji
		want  int
	}{
		{"loaded", loaded, 4},
		{"not loaded", customEmoji{emoji: discord.Emoji{ID: 2}, name: "dog"}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := test.emoji.Line(images)
			if len(got) != test.want || got[len(got)-1].Text != test.emoji.name {
				t.Errorf("Line = %v, want %d spans ending with the name", got, test.want)
			}
		})
	}
}

func TestUsableEmojis(t *testing.T) {
	const (
		guild discord.GuildID = 1
		other discord.GuildID = 2
		role  discord.RoleID  = 3
	)

	emojis := map[discord.GuildID][]discord.Emoji{
		guild: {
			{ID: 10, Name: "still"},
			{ID: 11, Name: "moving", Animated: true},
			{ID: 12, Name: "mine", RoleIDs: []discord.RoleID{role}},
			{ID: 13, Name: "theirs", RoleIDs: []discord.RoleID{4}},
		},
		other: {{ID: 20, Name: "elsewhere"}},
	}

	for _, test := range []struct {
		name  string
		nitro discord.UserNitro
		gID   discord.GuildID
		want  []discord.EmojiID
	}{
		{"without nitro", discord.NoUserNitro, guild, []discord.EmojiID{10, 12}},
		{"with nitro", discord.NitroFull, guild, []discord.EmojiID{10, 11, 12, 20}},
		{"without nitro in dms", discord.NoUserNitro, discord.NullGuildID, nil},
	} {
		t.Run(test.name, func(t *testing.T) {
			s := ningen.FromState(state.NewWithStore("", defaultstore.New()))
			if err := s.MyselfSet(discord.User{ID: 100, Nitro: test.nitro}, false); err != nil {
				t.Fatal(err)
			}

			for gID, guildEmojis := range emojis {
				if err := s.GuildSet(&discord.Guild{ID: gID}, false); err != nil {
					t.Fatal(err)
				}

				if err := s.EmojiSet(gID, guildEmojis, false); err != nil {
					t.Fatal(err)
				}

				member := discord.Member{User: discord.User{ID: 100}, RoleIDs: []discord.RoleID{role}}
				if err := s.MemberSet(gID, &member, false); err != nil {
					t.Fatal(err)
				}
			}

			var got []discord.EmojiID
			for _, emoji := range usableEmojis(s, test.gID) {
				got = append(got, emoji.ID)
			}

			if !slices.Equal(got, test.want) {
				t.Errorf("got %v, want %v", got, test.want)
			}
		})
	}
}
