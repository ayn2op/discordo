package guildstree

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
	"github.com/ayn2op/ningen/v3/states/read"
	"github.com/ayn2op/tview/tree"
)

func TestModelMoveDMToFront(t *testing.T) {
	t.Run("moves rendered channel to front", func(t *testing.T) {
		firstID := discord.ChannelID(1)
		secondID := discord.ChannelID(2)
		thirdID := discord.ChannelID(3)

		first := tree.NewNode("first").SetReference(firstID)
		second := tree.NewNode("second").SetReference(secondID)
		third := tree.NewNode("third").SetReference(thirdID)
		dmRoot := tree.NewNode("Direct Messages").SetChildren([]*tree.Node{first, second, third})
		m := Model{
			dmRootNode: dmRoot,
			channelNodes: map[discord.ChannelID]*tree.Node{
				firstID:  first,
				secondID: second,
				thirdID:  third,
			},
		}

		m.moveDMToFront(thirdID)

		want := []*tree.Node{third, first, second}
		if !slices.Equal(dmRoot.Children(), want) {
			t.Fatalf("children = %v, want %v", dmRoot.Children(), want)
		}
	})

	t.Run("ignores unrendered channel", func(t *testing.T) {
		firstID := discord.ChannelID(1)
		first := tree.NewNode("first").SetReference(firstID)
		dmRoot := tree.NewNode("Direct Messages").SetChildren([]*tree.Node{first})
		m := Model{
			dmRootNode:   dmRoot,
			channelNodes: map[discord.ChannelID]*tree.Node{},
		}

		m.moveDMToFront(discord.ChannelID(2))

		children := dmRoot.Children()
		if len(children) != 1 || children[0] != first {
			t.Fatalf("children = %v, want the original DM node", children)
		}
	})
}

func TestAdjacentNode(t *testing.T) {
	a, b, c := tree.NewNode("a"), tree.NewNode("b"), tree.NewNode("c")
	root := tree.NewNode("").SetChildren([]*tree.Node{a, tree.NewNode("guild").AddChild(b), c})
	match := func(n *tree.Node) bool { return n == a || n == b }
	for _, test := range []struct {
		name          string
		current       *tree.Node
		after, before *tree.Node
	}{
		{"none", nil, a, b},
		{"a", a, b, b},
		{"b", b, a, a},
		{"c", c, a, b},
	} {
		t.Run(test.name, func(t *testing.T) {
			t.Run("after", func(t *testing.T) {
				if got := adjacentNode(root, test.current, match, false); got != test.after {
					t.Errorf("got %v, want %v", got, test.after)
				}
			})
			t.Run("before", func(t *testing.T) {
				if got := adjacentNode(root, test.current, match, true); got != test.before {
					t.Errorf("got %v, want %v", got, test.before)
				}
			})
		})
	}
	t.Run("no match", func(t *testing.T) {
		if got := adjacentNode(root, a, func(*tree.Node) bool { return false }, false); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

func TestMentionsLabel(t *testing.T) {
	for _, test := range []struct {
		name  string
		count int
		want  string
	}{
		{"none", 0, ""},
		{"some", 5, " (5)"},
		{"most counted", maxMentions, " (99)"},
		{"more", maxMentions + 1, " (99+)"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if got := mentionsLabel(test.count); got != test.want {
				t.Errorf("mentionsLabel(%d) = %q, want %q", test.count, got, test.want)
			}
		})
	}
}

func TestModelMentions(t *testing.T) {
	s := ningen.FromState(state.NewWithStore("", defaultstore.New()))
	s.State.Call(&gateway.ReadyEvent{ReadStates: []gateway.ReadState{
		{ChannelID: 1, LastMessageID: 10, MentionCount: 2},
		{ChannelID: 2, LastMessageID: 20, MentionCount: 3},
	}})
	m := Model{state: s}
	channels := []discord.Channel{{ID: 1}, {ID: 2}, {ID: 3}}
	if got := m.mentions(channels...); got != 5 {
		t.Errorf("mentions = %d, want 5", got)
	}
}

func TestModelRefreshReadState(t *testing.T) {
	t.Run("keeps the name of a channel sharing its guild's ID", func(t *testing.T) {
		s := ningen.FromState(state.NewWithStore("", defaultstore.New()))
		me := discord.User{ID: 2}
		guild := discord.Guild{ID: 1, Name: "guild", OwnerID: me.ID}
		channel := discord.Channel{ID: 1, GuildID: guild.ID, Type: discord.GuildText, Name: "general"}
		if err := errors.Join(
			s.MyselfSet(me, false),
			s.GuildSet(&guild, false),
			s.MemberSet(guild.ID, &discord.Member{User: me}, false),
			s.RoleSet(guild.ID, &discord.Role{ID: discord.RoleID(guild.ID)}, false),
			s.ChannelSet(&channel, false),
		); err != nil {
			t.Fatal(err)
		}
		cfg := config.Default()
		m := NewModel(&cfg, s)
		m.createGuildNode(m.root, guild)
		guildNode := m.root.Children()[0]
		m.createChannelNode(guildNode, channel)
		channelNode := guildNode.Children()[0]

		m.refreshReadState(&read.UpdateEvent{ReadState: gateway.ReadState{ChannelID: 2}, GuildID: guild.ID})

		if got, want := guildNode.Spans()[0].Text, m.guildText(guild); got != want {
			t.Errorf("guild node text = %q, want %q", got, want)
		}
		if got, want := channelNode.Spans()[0].Text, m.channelText(channel); got != want {
			t.Errorf("channel node text = %q, want %q", got, want)
		}
	})
}
