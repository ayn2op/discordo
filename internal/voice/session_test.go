//go:build !no_voice

package voice

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
)

func TestSessionFollow(t *testing.T) {
	const (
		userID    discord.UserID    = 1
		guildID   discord.GuildID   = 2
		channelID discord.ChannelID = 3
	)

	// joined returns a Session that has joined the voice channel channelID of guildID.
	joined := func(t *testing.T) *Session {
		t.Helper()

		state := state.NewWithStore("", defaultstore.New())
		if err := state.Cabinet.MyselfSet(discord.User{ID: userID}, false); err != nil {
			t.Fatal(err)
		}
		if err := state.Cabinet.ChannelSet(&discord.Channel{ID: channelID, GuildID: guildID, Type: discord.GuildVoice}, false); err != nil {
			t.Fatal(err)
		}

		session := New(state, 0, 100)
		session.setStatus(Status{ChannelID: channelID, Joined: true})
		return session
	}

	tests := []struct {
		name  string
		event discord.VoiceState
		want  Status
	}{
		{
			name:  "moved",
			event: discord.VoiceState{GuildID: guildID, ChannelID: 4, UserID: userID},
			want:  Status{ChannelID: 4, Joined: true},
		},
		{
			name:  "disconnected",
			event: discord.VoiceState{GuildID: guildID, ChannelID: discord.NullChannelID, UserID: userID},
			want:  Status{},
		},
		{
			name:  "connected from another client",
			event: discord.VoiceState{GuildID: guildID, ChannelID: channelID, UserID: userID, SessionID: "other"},
			want:  Status{},
		},
		{
			// Joining a voice channel of one guild leaves that of another.
			name:  "left another guild",
			event: discord.VoiceState{GuildID: 5, ChannelID: discord.NullChannelID, UserID: userID},
			want:  Status{ChannelID: channelID, Joined: true},
		},
		{
			name:  "another user",
			event: discord.VoiceState{GuildID: guildID, ChannelID: discord.NullChannelID, UserID: 6},
			want:  Status{ChannelID: channelID, Joined: true},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			session := joined(t)
			session.follow(&gateway.VoiceStateUpdateEvent{VoiceState: test.event})

			if got := session.Status(); got != test.want {
				t.Fatalf("got %+v, want %+v", got, test.want)
			}
		})
	}
}
