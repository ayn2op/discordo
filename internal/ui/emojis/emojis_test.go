package emojis

import (
	"image"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
)

func TestAll(t *testing.T) {
	var names []string
	for name, emoji := range All(ningen.FromState(state.New("")), discord.NullGuildID) {
		if emoji.Name == "👍" {
			names = append(names, name)
		}
	}
	if !slices.Contains(names, "+1") || !slices.Contains(names, "thumbsup") {
		t.Errorf("👍 is named %v, want +1 and thumbsup among them", names)
	}
}

func TestLine(t *testing.T) {
	images := map[discord.URL]tviewimage.Widget{URLFromID(1): tviewimage.New(image.NewAlpha(image.Rect(0, 0, 2, 2))).Width(2)}
	for _, test := range []struct {
		name  string
		emoji discord.Emoji
		want  int
	}{
		{"loaded", discord.Emoji{ID: 1, Name: "cat"}, 4},
		{"not loaded", discord.Emoji{ID: 2, Name: "dog"}, 1},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := Line(test.emoji, test.emoji.Name, tview.Style{}, images)
			if len(got) != test.want || got[len(got)-1].Text != test.emoji.Name {
				t.Errorf("Line = %v, want %d spans ending with the name", got, test.want)
			}
		})
	}
	t.Run("unicode", func(t *testing.T) {
		if got := Line(discord.Emoji{Name: "👍"}, "thumbsup", tview.Style{}, nil); len(got) != 1 || got[0].Text != "👍 thumbsup" {
			t.Errorf("Line = %v, want the emoji before its name", got)
		}
	})
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
