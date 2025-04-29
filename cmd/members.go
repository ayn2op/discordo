package cmd

import (
	"context"
	"log/slog"
	"sync"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

type GuildMember struct {
	guild discord.GuildID
	user  discord.UserID
}

type GuildMembers struct {
	alreadyFetched map[GuildMember]bool
	fetchingChunk  bool
	done           chan struct{}
	mu             sync.Mutex
}

func newGuildMembers() *GuildMembers {
	gmState := GuildMembers{}
	gmState.alreadyFetched = make(map[GuildMember]bool)

	return &gmState
}

func (gm *GuildMembers) fetchChannelMembers(ms []discord.Message, requestPresences bool) {
	if len(ms) == 0 {
		return
	}

	guildID := ms[0].GuildID
	if !guildID.IsValid() {
		return
	}

	var toFetch []discord.UserID
	for _, m := range ms {
		if gm.alreadyFetched[GuildMember{guildID, m.Author.ID}] {
			continue
		}

		member, _ := discordState.Cabinet.Member(guildID, m.Author.ID)
		if member == nil {
			toFetch = append(toFetch, m.Author.ID)
		}

		gm.alreadyFetched[GuildMember{guildID, m.Author.ID}] = true
	}

	if toFetch == nil {
		return
	}

	err := discordState.Gateway().Send(context.Background(), &gateway.RequestGuildMembersCommand{
		GuildIDs:  []discord.GuildID{guildID},
		UserIDs:   toFetch,
		Presences: requestPresences,
	})
	if err != nil {
		for _, id := range toFetch {
			gm.alreadyFetched[GuildMember{guildID, id}] = false
		}

		slog.Error("Failed to request guild members", "err", err)
		return
	}

	gm.setFetchingChunk(true)
	gm.waitForChunkEvent()
}

func (gm *GuildMembers) setFetchingChunk(value bool) {
	gm.mu.Lock()
	gm.fetchingChunk = value
	gm.mu.Unlock()

	if value {
		gm.done = make(chan struct{})
	} else {
		close(gm.done)
	}
}

func (gm *GuildMembers) waitForChunkEvent() {
	gm.mu.Lock()
	if !gm.fetchingChunk {
		gm.mu.Unlock()
		return
	}
	gm.mu.Unlock()

	select {
	case <-gm.done:
	default:
		<-gm.done
	}
}
