package mentions

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/ningen/v3"
)

func TestItems(t *testing.T) {
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
		if got := len(Items(s, &channel, "", 10)); got != 2 {
			t.Errorf("got %d authors, want the 2 others", got)
		}
	})

	t.Run("limit", func(t *testing.T) {
		if got := len(Items(s, &channel, "", 1)); got != 1 {
			t.Errorf("got %d authors, want 1", got)
		}
	})

	t.Run("recipients", func(t *testing.T) {
		items := Items(s, &channel, "bob", 10)
		if len(items) != 1 || items[0].Insert() != "@bob" {
			t.Errorf("got %v, want @bob", items)
		}
	})
}
