package composer

import (
	"maps"
	"path/filepath"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/arikawa/v3/utils/sendpart"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	ningenEmoji "github.com/ayn2op/ningen/v3/states/emoji"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/gdamore/tcell/v3"
)

// newTestModel returns an enabled composer with the default config.
func newTestModel(t *testing.T) (*Model, *config.Config) {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	c := NewModel(cfg, ningen.FromState(state.New("")))
	c.disabled = false
	return &c, cfg
}

func TestModelSetChannel(t *testing.T) {
	// Everyone in guild 1 may send in threads, but not in channels.
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}

	state := ningen.FromState(state.NewWithStore("", defaultstore.New()))
	channel := discord.Channel{ID: 10, GuildID: 1}
	thread := discord.Channel{ID: 11, GuildID: 1, ParentID: 10, Type: discord.GuildPublicThread}
	for _, err := range []error{
		state.MyselfSet(discord.User{ID: 3}, false),
		state.GuildSet(&discord.Guild{ID: 1}, false),
		state.MemberSet(1, &discord.Member{User: discord.User{ID: 3}}, false),
		state.RoleSet(1, &discord.Role{ID: 1, Permissions: discord.PermissionSendMessagesInThreads}, false),
		state.ChannelSet(&channel, false),
		state.ChannelSet(&thread, false),
	} {
		if err != nil {
			t.Fatal(err)
		}
	}

	c := NewModel(cfg, state)
	for name, tt := range map[string]struct {
		channel  discord.Channel
		disabled bool
	}{
		"a channel that cannot be sent in disables it": {channel, true},
		"a thread that can be sent in enables it":      {thread, false},
	} {
		t.Run(name, func(t *testing.T) {
			if c.SetChannel(&tt.channel); c.Disabled() != tt.disabled {
				t.Fatalf("disabled = %v, want %v", c.Disabled(), tt.disabled)
			}
		})
	}
}

func TestModelUpdate(t *testing.T) {
	up := tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
	for _, tt := range []struct {
		name  string
		setup func(*Model, *config.Config) tview.KeyMsg
		want  bool
	}{
		{"empty", func(*Model, *config.Config) tview.KeyMsg { return up }, true},
		{"draft", func(c *Model, _ *config.Config) tview.KeyMsg { c.setText("draft", len("draft")); return up }, false},
		{"reply", func(c *Model, _ *config.Config) tview.KeyMsg {
			c.StartReply(discord.Message{ID: 3}, "name", false)
			return up
		}, false},
		{"attachment", func(c *Model, _ *config.Config) tview.KeyMsg {
			c.sendMessageData.Files = []sendpart.File{{Name: "file"}}
			return up
		}, false},
		{"editing", func(c *Model, _ *config.Config) tview.KeyMsg { c.StartEdit(discord.Message{}); return up }, false},
		{"disabled", func(c *Model, _ *config.Config) tview.KeyMsg { c.disabled = true; return up }, false},
		{"remapped", func(_ *Model, cfg *config.Config) tview.KeyMsg {
			cfg.Keybinds.Composer.EditLast.Keybind = keybind.New("ctrl+p")
			return tcell.NewEventKey(tcell.KeyCtrlP, "", tcell.ModNone)
		}, true},
		{"unbound", func(_ *Model, cfg *config.Config) tview.KeyMsg {
			cfg.Keybinds.Composer.EditLast.Keybind = keybind.New()
			return up
		}, false},
	} {
		t.Run("edit last message: "+tt.name, func(t *testing.T) {
			c, cfg := newTestModel(t)
			// setup changes c, so it runs before Update copies it.
			msg := tt.setup(c, cfg)
			var cmd tview.Cmd
			*c, cmd = c.Update(msg)
			if got := cmd != nil && cmd() == (EditLastMsg{}); got != tt.want {
				t.Fatalf("asked to edit the last message = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelHandle(t *testing.T) {
	c, _ := newTestModel(t)
	area := tview.Rectangle{Width: 20, Height: 3}

	t.Run("typing edits the text", func(t *testing.T) {
		msg := c.View(true).Handle(tcell.NewEventKey(tcell.KeyRune, "a", tcell.ModNone), area)
		if _, ok := msg.(editMsg); !ok {
			t.Fatalf("got %v, want an editMsg", msg)
		}
		*c, _ = c.Update(msg)
		if c.text != "a" {
			t.Fatalf("text = %q", c.text)
		}
	})
	t.Run("the newline keybind inserts a newline", func(t *testing.T) {
		msg := c.View(true).Handle(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModShift), area)
		if _, ok := msg.(editMsg); !ok {
			t.Fatalf("got %v, want an editMsg", msg)
		}
		*c, _ = c.Update(msg)
		if c.text != "a\n" {
			t.Fatalf("text = %q", c.text)
		}
	})
	t.Run("bound keys are left to Update", func(t *testing.T) {
		enter := tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone)
		if got := c.View(true).Handle(enter, area); got != enter {
			t.Fatalf("got %v", got)
		}
	})
}

func TestModelSearchMember(t *testing.T) {
	c, _ := newTestModel(t)
	c.state.MemberState.SearchLimit = 2
	c.CacheMemberSearch(&gateway.GuildMembersChunkEvent{
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
	c.OnGuildMemberRemove(&gateway.GuildMemberRemoveEvent{
		GuildID: 1,
		User:    discord.User{Username: "abc"},
	})
	want := map[string]uint{"1 ab": 1, "1 abc": 1, "2 a": 2}
	if !maps.Equal(c.memberSearchCache, want) {
		t.Fatalf("cache after invalidation = %v, want %v", c.memberSearchCache, want)
	}
}

func TestModelTabSuggest(t *testing.T) {
	c, _ := newTestModel(t)
	if err := c.state.MyselfSet(discord.User{ID: 1, Username: "1"}, false); err != nil {
		t.Fatal(err)
	}
	channel := discord.Channel{ID: 1, Type: discord.DirectMessage}
	for id, author := range []discord.UserID{2, 3, 2, 1} {
		message := discord.Message{ID: discord.MessageID(id + 1), ChannelID: 1, Author: discord.User{ID: author, Username: author.String()}}
		if err := c.state.MessageSet(&message, false); err != nil {
			t.Fatal(err)
		}
	}
	c.SetChannel(&channel)
	c.setText("@", 1)
	c.tabSuggest()
	if got := c.mentionsList.ItemCount(); got != 2 {
		t.Fatalf("suggested %d authors, want 2", got)
	}
}

func TestModelProcessText(t *testing.T) {
	c, _ := newTestModel(t)
	channel := &discord.Channel{Type: discord.DirectMessage, DMRecipients: []discord.User{{ID: 1, Username: "alice"}, {ID: 2, Username: "bob"}}}
	for _, tt := range []struct{ name, in, want string }{
		{"mentions around code", "@alice `code` @bob", "<@1> `code` <@2>"},
		{"mention in code", "@alice `@bob`", "<@1> `@bob`"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if got := c.processText(channel, []byte(tt.in)); got != tt.want {
				t.Fatalf("got %q, want %q", got, tt.want)
			}
		})
	}
}

func TestModelEmojiSuggest(t *testing.T) {
	c, _ := newTestModel(t)
	channel := discord.Channel{ID: 1, Type: discord.DirectMessage}
	c.SetChannel(&channel)
	c.setText(":smi", 4)
	c.tabSuggest()
	if got := c.emojisList.ItemCount(); got == 0 {
		t.Fatal("expected emoji suggestions for ':smi', got none")
	}
	if !c.emojisListVisible() {
		t.Fatal("expected emojisListVisible to be true")
	}
	if c.mentionsListVisible() {
		t.Fatal("expected mentionsListVisible to be false")
	}
}

func TestModelEmojiComplete(t *testing.T) {
	c, _ := newTestModel(t)
	channel := discord.Channel{ID: 1, Type: discord.DirectMessage}
	c.SetChannel(&channel)
	c.setText("hello :smile", 12)
	c.tabSuggest()
	if !c.emojisListVisible() {
		t.Fatal("expected emojis list visible")
	}
	c.tabComplete()
	if got := c.text; got != "hello 😄 " {
		t.Fatalf("text after completion = %q, want 'hello 😄 '", got)
	}
	if c.emojisListVisible() {
		t.Fatal("expected emojis list to be hidden after completion")
	}
}

func TestModelCustomEmojiComplete(t *testing.T) {
	c, _ := newTestModel(t)
	guildID := discord.GuildID(100)
	channel := discord.Channel{ID: 10, GuildID: guildID, Type: discord.GuildText}
	c.state.EmojiState = ningenEmoji.NewState(c.state.Cabinet)
	_ = c.state.Cabinet.GuildSet(&discord.Guild{ID: guildID}, false)
	_ = c.state.Cabinet.EmojiSet(guildID, []discord.Emoji{
		{ID: 555, Name: "custom_pepe"},
	}, false)

	c.SetChannel(&channel)
	c.setText(":custom_pepe", 12)
	c.tabSuggest()
	if !c.emojisListVisible() {
		t.Fatal("expected emojis list visible")
	}
	c.tabComplete()
	if got := c.text; got != "<:custom_pepe:555> " {
		t.Fatalf("text after completion = %q, want '<:custom_pepe:555> '", got)
	}
	if c.emojisListVisible() {
		t.Fatal("expected emojis list to be hidden after completion")
	}
}
