package composer

import (
	"maps"
	"path/filepath"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/utils/sendpart"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
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
	return c, cfg
}

func TestModelUpdate(t *testing.T) {
	up := tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone)
	for _, tt := range []struct {
		name  string
		setup func(*Model, *config.Config) tview.KeyMsg
		want  bool
	}{
		{"empty", func(*Model, *config.Config) tview.KeyMsg { return up }, true},
		{"draft", func(c *Model, _ *config.Config) tview.KeyMsg { c.setText("draft"); return up }, false},
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
			cmd := c.Update(tt.setup(c, cfg))
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
		c.Update(msg)
		if c.editState.Value() != "a" {
			t.Fatalf("text = %q", c.editState.Value())
		}
	})
	t.Run("the newline keybind inserts a newline", func(t *testing.T) {
		msg := c.View(true).Handle(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModShift), area)
		if _, ok := msg.(editMsg); !ok {
			t.Fatalf("got %v, want an editMsg", msg)
		}
		c.Update(msg)
		if c.editState.Value() != "a\n" {
			t.Fatalf("text = %q", c.editState.Value())
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
	if err := c.state.Cabinet.MyselfSet(discord.User{ID: 1, Username: "1"}, false); err != nil {
		t.Fatal(err)
	}
	channel := discord.Channel{ID: 1, Type: discord.DirectMessage}
	for id, author := range []discord.UserID{2, 3, 2, 1} {
		message := discord.Message{ID: discord.MessageID(id + 1), ChannelID: 1, Author: discord.User{ID: author, Username: author.String()}}
		if err := c.state.Cabinet.MessageSet(&message, false); err != nil {
			t.Fatal(err)
		}
	}
	c.SetChannel(&channel)
	c.setText("@")
	c.tabSuggest()
	if got := c.mentionsList.ItemCount(); got != 2 {
		t.Fatalf("suggested %d authors, want 2", got)
	}
}
