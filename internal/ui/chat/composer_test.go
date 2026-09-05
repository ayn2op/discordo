package chat

import (
	"maps"
	"path/filepath"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/config"
)

func TestMemberSearchCache(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg, "")
	m.state.MemberState.SearchLimit = 2
	c := m.composer
	c.onGuildMembersChunk(&gateway.GuildMembersChunkEvent{
		Nonce:   memberSearchNonce + "1 ab",
		Members: []discord.Member{{}},
	})
	if cmd := c.searchMember(1, "ab"); cmd != nil {
		t.Fatal("cached query returned a network command")
	}
	if cmd := c.searchMember(1, "abc"); cmd != nil || c.memberSearchCache["1 abc"] != 1 {
		t.Fatal("incomplete prefix results were not reused")
	}
	c.memberSearchCache["1 a"] = 2
	c.memberSearchCache["2 a"] = 2
	m.onGuildMemberRemove(&gateway.GuildMemberRemoveEvent{
		GuildID: 1,
		User:    discord.User{Username: "abc"},
	})
	want := map[string]uint{"1 ab": 1, "1 abc": 1, "2 a": 2}
	if !maps.Equal(c.memberSearchCache, want) {
		t.Fatalf("cache after invalidation = %v, want %v", c.memberSearchCache, want)
	}
}
