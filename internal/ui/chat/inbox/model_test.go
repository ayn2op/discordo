package inbox

import (
	"errors"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
)

// guildID is the guild the test channels are in, whose name follows theirs in a row.
const guildID = discord.GuildID(1)

// newTestModel returns a model over a state with channels in one guild and the read states given.
func newTestModel(t *testing.T, channels []discord.Channel, readStates []gateway.ReadState) Model {
	t.Helper()

	s := ningen.FromState(state.NewWithStore("", defaultstore.New()))
	me := discord.User{ID: 2}
	guild := discord.Guild{ID: guildID, Name: "guild", OwnerID: me.ID}
	errs := []error{
		s.MyselfSet(me, false),
		s.GuildSet(&guild, false),
		s.MemberSet(guild.ID, &discord.Member{User: me}, false),
		s.RoleSet(guild.ID, &discord.Role{ID: discord.RoleID(guild.ID)}, false),
	}
	for _, channel := range channels {
		errs = append(errs, s.ChannelSet(&channel, false))
	}
	if err := errors.Join(errs...); err != nil {
		t.Fatal(err)
	}

	s.State.Call(&gateway.ReadyEvent{ReadStates: readStates})
	cfg := config.Default()
	return NewModel(&cfg, s)
}

// channel returns a text channel of the test guild whose newest message is lastMessageID.
func channel(id discord.ChannelID, name string, lastMessageID discord.MessageID) discord.Channel {
	return discord.Channel{ID: id, GuildID: guildID, Type: discord.GuildText, Name: name, LastMessageID: lastMessageID}
}

func TestModelRefresh(t *testing.T) {
	t.Run("lists the unread channels, mentions first", func(t *testing.T) {
		m := newTestModel(t,
			[]discord.Channel{channel(1, "first", 30), channel(2, "second", 20)},
			[]gateway.ReadState{{ChannelID: 1, LastMessageID: 29}, {ChannelID: 2, LastMessageID: 19, MentionCount: 2}},
		)

		m.Refresh()

		want := []discord.ChannelID{2, 1}
		if got := channelIDs(m.entries); !slices.Equal(got, want) {
			t.Errorf("entries = %v, want %v", got, want)
		}
	})

	t.Run("lists the channel written to last first among those without mentions", func(t *testing.T) {
		m := newTestModel(t,
			[]discord.Channel{channel(1, "older", 20), channel(2, "newer", 30)},
			[]gateway.ReadState{{ChannelID: 1, LastMessageID: 19}, {ChannelID: 2, LastMessageID: 29}},
		)

		m.Refresh()

		want := []discord.ChannelID{2, 1}
		if got := channelIDs(m.entries); !slices.Equal(got, want) {
			t.Errorf("entries = %v, want %v", got, want)
		}
	})

	t.Run("leaves out a channel read and one never read from", func(t *testing.T) {
		m := newTestModel(t,
			[]discord.Channel{channel(1, "read", 20), channel(2, "unknown", 30), channel(3, "unread", 40)},
			[]gateway.ReadState{{ChannelID: 1, LastMessageID: 20}, {ChannelID: 3, LastMessageID: 39}},
		)

		m.Refresh()

		want := []discord.ChannelID{3}
		if got := channelIDs(m.entries); !slices.Equal(got, want) {
			t.Errorf("entries = %v, want %v", got, want)
		}
	})

	t.Run("names the guild of a channel and counts its mentions", func(t *testing.T) {
		m := newTestModel(t,
			[]discord.Channel{channel(1, "general", 20)},
			[]gateway.ReadState{{ChannelID: 1, LastMessageID: 19, MentionCount: 3}},
		)

		m.Refresh()

		if len(m.entries) != 1 {
			t.Fatalf("entries = %v, want one", m.entries)
		}
		if got, want := m.entries[0].text, "#general - guild (3)"; got != want {
			t.Errorf("text = %q, want %q", got, want)
		}
	})

	t.Run("keeps the message read last, which opening the channel acknowledges", func(t *testing.T) {
		m := newTestModel(t,
			[]discord.Channel{channel(1, "general", 20)},
			[]gateway.ReadState{{ChannelID: 1, LastMessageID: 15}},
		)

		m.Refresh()

		if len(m.entries) != 1 {
			t.Fatalf("entries = %v, want one", m.entries)
		}
		if got, want := m.entries[0].lastReadID, discord.MessageID(15); got != want {
			t.Errorf("last read = %d, want %d", got, want)
		}
	})

	t.Run("highlights the first row listed", func(t *testing.T) {
		m := newTestModel(t,
			[]discord.Channel{channel(1, "first", 30), channel(2, "second", 20)},
			[]gateway.ReadState{{ChannelID: 1, LastMessageID: 29}, {ChannelID: 2, LastMessageID: 19}},
		)

		m.Refresh()

		item, ok := m.Selected()
		if !ok {
			t.Fatal("no row is highlighted")
		}
		if got, want := item.Reference, m.entries[0]; got != want {
			t.Errorf("highlighted = %v, want %v", got, want)
		}
	})
}

func TestModelMarkRead(t *testing.T) {
	t.Run("drops the row of the channel acknowledged", func(t *testing.T) {
		m := newTestModel(t,
			[]discord.Channel{channel(1, "first", 30), channel(2, "second", 20)},
			[]gateway.ReadState{{ChannelID: 1, LastMessageID: 29}, {ChannelID: 2, LastMessageID: 19}},
		)
		m.Refresh()

		if cmd := m.markRead(); cmd == nil {
			t.Error("no command acknowledges the channel")
		}

		want := []discord.ChannelID{2}
		if got := channelIDs(m.entries); !slices.Equal(got, want) {
			t.Errorf("entries = %v, want %v", got, want)
		}
	})

	t.Run("does nothing with an empty inbox", func(t *testing.T) {
		m := newTestModel(t, nil, nil)
		m.Refresh()

		if cmd := m.markRead(); cmd != nil {
			t.Error("a command acknowledges a channel that is not listed")
		}
	})
}

func TestModelMarkAllRead(t *testing.T) {
	m := newTestModel(t,
		[]discord.Channel{channel(1, "first", 30), channel(2, "second", 20)},
		[]gateway.ReadState{{ChannelID: 1, LastMessageID: 29}, {ChannelID: 2, LastMessageID: 19}},
	)
	m.Refresh()

	if cmd := m.markAllRead(); cmd == nil {
		t.Error("no command acknowledges the channels")
	}

	if len(m.entries) != 0 {
		t.Errorf("entries = %v, want none", m.entries)
	}
}

// channelIDs returns the channel of each entry, in order.
func channelIDs(entries []entry) []discord.ChannelID {
	ids := make([]discord.ChannelID, len(entries))
	for index, e := range entries {
		ids[index] = e.channelID
	}
	return ids
}
