package memberstree

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview/tree"
)

const guildID, channelID = 1, 2

func newTestModel(t *testing.T) Model {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	return NewModel(cfg, ningen.FromState(state.New("")))
}

// setGuildMembers stores a guild channel whose member list holds items, given as JSON.
func setGuildMembers(t *testing.T, m Model, items string) *discord.Channel {
	t.Helper()
	channel := &discord.Channel{ID: channelID, GuildID: guildID, Type: discord.GuildText}
	m.state.Cabinet.ChannelSet(channel, false)
	m.state.Cabinet.RoleSet(guildID, &discord.Role{ID: 3, Name: "Mods"}, false)

	event := &gateway.GuildMemberListUpdateEvent{ID: "everyone", GuildID: guildID}
	event.Ops = []gateway.GuildMemberListOp{{Op: "SYNC", Range: [2]int{0, 99}}}
	if err := json.Unmarshal([]byte(items), &event.Ops[0].Items); err != nil {
		t.Fatal(err)
	}
	m.state.State.Handler.Call(event)
	return channel
}

// lines returns the text of the groups and, indented, their members.
func lines(m Model) []string {
	var lines []string
	for _, group := range m.root.Children() {
		lines = append(lines, text(group))
		for _, member := range group.Children() {
			lines = append(lines, "  "+text(member))
		}
	}
	return lines
}

func text(node *tree.Node) string {
	var s string
	for _, segment := range node.Line() {
		s += segment.Text
	}
	return s
}

func TestModelShown(t *testing.T) {
	for _, tt := range []struct {
		name    string
		channel *discord.Channel
		want    bool
	}{
		{"none", nil, false},
		{"guild channel", &discord.Channel{GuildID: guildID, Type: discord.GuildText}, true},
		{"thread", &discord.Channel{GuildID: guildID, Type: discord.GuildPublicThread}, false},
		{"group dm", &discord.Channel{Type: discord.GroupDM}, true},
		{"dm", &discord.Channel{Type: discord.DirectMessage}, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t)
			m.channel = tt.channel
			if got := m.Shown(); got != tt.want {
				t.Fatalf("Shown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelRebuild(t *testing.T) {
	t.Run("guild members", func(t *testing.T) {
		m := newTestModel(t)
		m.channel = setGuildMembers(t, m, `[
			{"group": {"id": "3", "count": 1}},
			{"member": {"user": {"id": "10", "username": "a"}, "nick": "Alice"}},
			{"group": {"id": "online", "count": 2}},
			{"member": {"user": {"id": "11", "username": "b"}}},
			{},
			{"member": {"user": {"id": "12", "username": "c"}}}
		]`)
		m.rebuild()

		// The member after the unloaded item belongs to a group that has yet to arrive.
		want := []string{"Mods — 1", "  ● Alice", "Online — 2", "  ● b", "Members", "  ● c"}
		if got := lines(m); !slices.Equal(got, want) {
			t.Fatalf("lines = %q, want %q", got, want)
		}
	})

	t.Run("group dm recipients", func(t *testing.T) {
		m := newTestModel(t)
		m.state.Cabinet.MyselfSet(discord.User{ID: 13, Username: "me"}, false)
		m.state.Cabinet.PresenceSet(discord.NullGuildID, &discord.Presence{User: discord.User{ID: 11}, Status: discord.OnlineStatus}, false)
		m.channel = &discord.Channel{Type: discord.GroupDM, DMRecipients: []discord.User{
			{ID: 10, Username: "a"},
			{ID: 11, Username: "c"},
			{ID: 12, Username: "B"},
		}}
		m.rebuild()

		// Online members come first, then members by name, including the current user.
		want := []string{"Members — 4", "  ● c", "  ● a", "  ● B", "  ● me"}
		if got := lines(m); !slices.Equal(got, want) {
			t.Fatalf("lines = %q, want %q", got, want)
		}
	})

	t.Run("keeps selection and collapsed groups", func(t *testing.T) {
		m := newTestModel(t)
		m.channel = setGuildMembers(t, m, `[
			{"group": {"id": "online", "count": 1}},
			{"member": {"user": {"id": "10", "username": "a"}}},
			{"group": {"id": "offline", "count": 1}},
			{"member": {"user": {"id": "11", "username": "b"}}}
		]`)
		m.rebuild()
		online, offline := m.root.Children()[0], m.root.Children()[1]
		selected := online.Children()[0]
		m.selectionState.SetCurrentNode(selected)
		offline.Collapse()

		m.rebuild()
		if got := m.selectionState.CurrentNode(); got != selected || m.root.PathTo(got) == nil {
			t.Fatal("the selected member was not kept in the tree")
		}
		if m.root.Children()[1].Expanded() {
			t.Fatal("the collapsed group was expanded")
		}
	})
}
