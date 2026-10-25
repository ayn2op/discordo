package composer

import (
	"image"
	"maps"
	"path/filepath"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/state/store/defaultstore"
	"github.com/ayn2op/arikawa/v3/utils/sendpart"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
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

	first := discord.Channel{ID: 1, Type: discord.DirectMessage}
	second := discord.Channel{ID: 2, Type: discord.DirectMessage}

	t.Run("drops the reply, keeping the text and files", func(t *testing.T) {
		c, _ := newTestModel(t)
		c.SetChannel(&first)
		c.StartReply(discord.Message{ID: 5}, "ann", true)
		c.setText("hi", 2)
		c.sendMessageData.Files = []sendpart.File{{Name: "a.png"}}

		c.SetChannel(&second)
		if c.sendMessageData.Reference != nil || c.title() != "" {
			t.Errorf("still replying: title %q", c.title())
		}

		if c.text != "hi" || len(c.sendMessageData.Files) != 1 {
			t.Errorf("text %q and %d files, want the text and file kept", c.text, len(c.sendMessageData.Files))
		}
	})

	t.Run("drops the edit and its text", func(t *testing.T) {
		c, _ := newTestModel(t)
		c.SetChannel(&first)
		c.StartEdit(discord.Message{ID: 5, Content: "old"})

		c.SetChannel(&second)
		if c.editing != nil || c.text != "" {
			t.Errorf("still editing %v with text %q", c.editing, c.text)
		}
	})

	t.Run("keeps the reply in the same channel", func(t *testing.T) {
		c, _ := newTestModel(t)
		c.SetChannel(&first)
		c.StartReply(discord.Message{ID: 5}, "ann", false)

		c.SetChannel(&first)
		if c.sendMessageData.Reference == nil {
			t.Error("reply dropped though the channel did not change")
		}
	})
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

	t.Run("images before completing", func(t *testing.T) {
		c, _ := newTestModel(t)
		c.Update(ui.ImagesUpdatedMsg{"cat": tviewimage.New(image.NewAlpha(image.Rect(0, 0, 2, 2)))})
	})
}

func TestModelHandle(t *testing.T) {
	c, _ := newTestModel(t)
	c.SetFocused(true)
	area := tview.Rectangle{Width: 20, Height: 3}

	t.Run("typing edits the text", func(t *testing.T) {
		msg := c.View().Handle(tcell.NewEventKey(tcell.KeyRune, "a", tcell.ModNone), area)
		if _, ok := msg.(editMsg); !ok {
			t.Fatalf("got %v, want an editMsg", msg)
		}
		*c, _ = c.Update(msg)
		if c.text != "a" {
			t.Fatalf("text = %q", c.text)
		}
	})
	t.Run("the newline keybind inserts a newline", func(t *testing.T) {
		msg := c.View().Handle(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModShift), area)
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
		if got := c.View().Handle(enter, area); got != enter {
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

func TestModelShowAutocomplete(t *testing.T) {
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
	c.showAutocomplete()
	if !c.autocomplete.Visible() {
		t.Fatal("no authors completed")
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

func TestModelQuery(t *testing.T) {
	c, _ := newTestModel(t)
	for _, test := range []struct {
		text string
		r    rune
		name string
	}{
		{":jo", ':', "jo"},
		{"hi :+1", ':', "+1"},
		{"hi :", ':', ""},
		{"@ab", '@', "ab"},
		{"hi @", '@', ""},
		{"12:30", 0, ""},
		{"a:joy", 0, ""},
		{"joy", 0, ""},
	} {
		t.Run(test.text, func(t *testing.T) {
			c.setText(test.text, len(test.text))
			if r, name := c.query(); r != test.r || name != test.name {
				t.Errorf("query() = %q, %q; want %q, %q", r, name, test.r, test.name)
			}
		})
	}
	t.Run("mentions disabled", func(t *testing.T) {
		c, cfg := newTestModel(t)
		cfg.UI.Autocomplete.Mentions.Enabled = false
		c.setText("@ab", len("@ab"))
		if trigger, _ := c.query(); trigger != 0 {
			t.Errorf("query() trigger = %q, want 0", trigger)
		}
	})

	t.Run("emojis disabled", func(t *testing.T) {
		c, cfg := newTestModel(t)
		cfg.UI.Autocomplete.Emojis.Enabled = false
		c.setText(":jo", len(":jo"))
		if trigger, _ := c.query(); trigger != 0 {
			t.Errorf("query() trigger = %q, want 0", trigger)
		}
	})
}

func TestModelComplete(t *testing.T) {
	c, _ := newTestModel(t)
	c.SetChannel(&discord.Channel{ID: 1, Type: discord.DirectMessage})
	c.setText("hi :joy", len("hi :joy"))
	c.showAutocomplete()
	if !c.autocomplete.Visible() {
		t.Fatal("no emojis completed")
	}

	c.complete()
	if want := "hi 😂 "; c.text != want || c.cursor != len(want) {
		t.Errorf("text = %q, cursor = %d; want %q, %d", c.text, c.cursor, want, len(want))
	}

	if c.autocomplete.Visible() {
		t.Error("autocomplete still shown after completing")
	}
}
