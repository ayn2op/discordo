package chat

import (
	"maps"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/utils/sendpart"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v3"
)

func TestComposerEditLastMessage(t *testing.T) {
	for _, scenario := range []string{"latest", "old", "draft", "multiline", "reply", "attachment", "editing", "disabled", "no own message", "remapped", "unbound"} {
		t.Run(scenario, func(t *testing.T) {
			cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
			if err != nil {
				t.Fatal(err)
			}
			m := NewModel(cfg, "")
			if err := m.state.Cabinet.MyselfSet(discord.User{ID: 1}, false); err != nil {
				t.Fatal(err)
			}
			m.SetSelectedChannel(&discord.Channel{ID: 1})
			c := m.composer
			c.SetDisabled(false)
			messages := []discord.Message{
				{ID: 1, ChannelID: 1, Author: discord.User{ID: 1}, Content: "old message"},
				{ID: 2, ChannelID: 1, Author: discord.User{ID: 1}, Content: "latest message"},
				{ID: 3, ChannelID: 1, Author: discord.User{ID: 2}, Content: "someone else"},
				{ID: 4, ChannelID: 1, Author: discord.User{ID: 1}, Type: discord.ChannelPinnedMessage, Content: "system"},
				{ID: 5, ChannelID: 1, Author: discord.User{ID: 1}},
			}
			want := ""
			key := tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
			switch scenario {
			case "latest":
				want = "latest message"
			case "old":
				messages = messages[:1]
				want = "old message"
			case "draft":
				c.SetText("draft", true)
			case "multiline":
				c.SetText("first line\nsecond line", true)
			case "reply":
				c.sendMessageData.Reference = &discord.MessageReference{MessageID: 3}
			case "attachment":
				c.sendMessageData.Files = []sendpart.File{{Name: "file"}}
			case "editing":
				c.edit = true
			case "disabled":
				c.SetDisabled(true)
			case "no own message":
				messages = messages[2:3]
			case "remapped":
				cfg.Keybinds.Composer.EditLast.SetKeys("ctrl+p")
				key = tcell.NewEventKey(tcell.KeyCtrlP, "", tcell.ModNone)
				want = "latest message"
			case "unbound":
				cfg.Keybinds.Composer.EditLast.SetKeys()
			}
			slices.Reverse(messages)
			m.messagesList.setMessages(messages)
			before := c.Text()
			wasEditing := c.edit
			c.Update(key)
			if want != "" {
				selected, ok := m.messagesList.selectedMessage()
				if !c.edit || c.Text() != want || !ok || selected.Content != want {
					t.Fatalf("edit = %v, text = %q, selected = %v", c.edit, c.Text(), selected)
				}
			} else if c.Text() != before || c.edit != wasEditing {
				t.Fatalf("composer changed: edit = %v, text = %q", c.edit, c.Text())
			}
		})
	}
}

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
