package guildstree

import (
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3"
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
			nodes: map[discord.Snowflake]*tree.Node{
				discord.Snowflake(firstID):  first,
				discord.Snowflake(secondID): second,
				discord.Snowflake(thirdID):  third,
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
			dmRootNode: dmRoot,
			nodes:      map[discord.Snowflake]*tree.Node{},
		}

		m.moveDMToFront(discord.ChannelID(2))

		children := dmRoot.Children()
		if len(children) != 1 || children[0] != first {
			t.Fatalf("children = %v, want the original DM node", children)
		}
	})
}

func TestCreateChannelNodeThreads(t *testing.T) {
	t.Run("adds a thread despite the view channel permission", func(t *testing.T) {
		cfg := config.Default()
		m := Model{cfg: &cfg, state: ningen.New(""), nodes: map[discord.Snowflake]*tree.Node{}}
		parent := tree.NewNode("parent").SetReference(discord.ChannelID(2))

		m.createChannelNode(parent, discord.Channel{ID: 3, GuildID: 1, ParentID: 2, Type: discord.GuildPublicThread, Name: "thread"})

		if got := len(parent.Children()); got != 1 {
			t.Fatalf("children = %v, want %v", got, 1)
		}
		if node := m.nodes[discord.Snowflake(discord.ChannelID(3))]; node == nil {
			t.Fatalf("node for thread = %v, want a node", node)
		}
	})

	t.Run("skips an archived thread", func(t *testing.T) {
		cfg := config.Default()
		m := Model{cfg: &cfg, state: ningen.New(""), nodes: map[discord.Snowflake]*tree.Node{}}
		parent := tree.NewNode("parent").SetReference(discord.ChannelID(2))

		m.createChannelNode(parent, discord.Channel{ID: 3, GuildID: 1, ParentID: 2, Type: discord.GuildPublicThread, Name: "thread", ThreadMetadata: &discord.ThreadMetadata{Archived: true}})

		if got := len(parent.Children()); got != 0 {
			t.Fatalf("children = %v, want %v", got, 0)
		}
	})
}

func TestCreateChannelNodesThreadBeforeParent(t *testing.T) {
	const (
		categoryID discord.ChannelID = 3
		parentID   discord.ChannelID = 4
		threadID   discord.ChannelID = 5
	)

	category := discord.Channel{ID: categoryID, GuildID: testGuildID, Type: discord.GuildCategory, Name: "category"}
	parent := discord.Channel{ID: parentID, GuildID: testGuildID, ParentID: categoryID, Type: discord.GuildText, Name: "parent"}
	thread := discord.Channel{ID: threadID, GuildID: testGuildID, ParentID: parentID, Type: discord.GuildPublicThread, Name: "thread"}

	cfg := config.Default()
	m := Model{cfg: &cfg, state: testGuildState(t, parent), nodes: map[discord.Snowflake]*tree.Node{}}

	m.createChannelNodes(tree.NewNode(""), []discord.Channel{category, thread, parent})

	parentNode := channelNode(t, m, parentID)
	threadNode := channelNode(t, m, threadID)
	if got := parentNode.Children(); len(got) != 1 || got[0] != threadNode {
		t.Fatalf("parent children = %v, want %v", got, []*tree.Node{threadNode})
	}
}

func TestModelExpandable(t *testing.T) {
	t.Run("a channel holding thread children is not expandable", func(t *testing.T) {
		channel := discord.Channel{ID: 3, GuildID: testGuildID, Type: discord.GuildText, Name: "channel"}
		cfg := config.Default()
		m := Model{cfg: &cfg, state: testGuildState(t, channel), nodes: map[discord.Snowflake]*tree.Node{}}
		m.createChannelNode(tree.NewNode(""), channel)

		node := channelNode(t, m, channel.ID)
		node.AddChild(tree.NewNode("thread"))

		if got := m.expandable(node); got {
			t.Fatalf("expandable = %v, want %v", got, false)
		}
	})

	t.Run("a category with a child is expandable", func(t *testing.T) {
		category := discord.Channel{ID: 3, GuildID: testGuildID, Type: discord.GuildCategory, Name: "category"}
		cfg := config.Default()
		m := Model{cfg: &cfg, state: testGuildState(t), nodes: map[discord.Snowflake]*tree.Node{}}
		m.createChannelNode(tree.NewNode(""), category)

		node := channelNode(t, m, category.ID)
		node.AddChild(tree.NewNode("channel"))

		if got := m.expandable(node); !got {
			t.Fatalf("expandable = %v, want %v", got, true)
		}
	})

	t.Run("a voice channel holding member children is not expandable", func(t *testing.T) {
		channel := discord.Channel{ID: 3, GuildID: testGuildID, Type: discord.GuildVoice, Name: "voice"}
		cfg := config.Default()
		m := Model{cfg: &cfg, state: testGuildState(t, channel), nodes: map[discord.Snowflake]*tree.Node{}}
		m.createChannelNode(tree.NewNode(""), channel)

		node := channelNode(t, m, channel.ID)
		node.AddChild(tree.NewNode("member"))

		if got := m.expandable(node); got {
			t.Fatalf("expandable = %v, want %v", got, false)
		}
	})
}

func TestAdjacentNode(t *testing.T) {
	a, b, c := tree.NewNode("a"), tree.NewNode("b"), tree.NewNode("c")
	root := tree.NewNode("").SetChildren([]*tree.Node{a, tree.NewNode("guild").AddChild(b), c})
	match := func(n *tree.Node) bool { return n == a || n == b }
	for current, want := range map[*tree.Node][2]*tree.Node{nil: {a, b}, a: {b, b}, b: {a, a}, c: {a, b}} {
		if got := adjacentNode(root, current, match, false); got != want[0] {
			t.Errorf("after %v = %v, want %v", current, got, want[0])
		}
		if got := adjacentNode(root, current, match, true); got != want[1] {
			t.Errorf("before %v = %v, want %v", current, got, want[1])
		}
	}
	if got := adjacentNode(root, a, func(*tree.Node) bool { return false }, false); got != nil {
		t.Errorf("got %v, want nil", got)
	}
}

const (
	testGuildID discord.GuildID = 1
	testUserID  discord.UserID  = 2
)

func testGuildState(t *testing.T, channels ...discord.Channel) *ningen.State {
	t.Helper()

	state := ningen.New("")
	if err := state.Cabinet.MyselfSet(discord.User{ID: testUserID}, false); err != nil {
		t.Fatal(err)
	}
	if err := state.Cabinet.GuildSet(&discord.Guild{ID: testGuildID}, false); err != nil {
		t.Fatal(err)
	}
	if err := state.Cabinet.RoleSet(testGuildID, &discord.Role{ID: discord.RoleID(testGuildID), Permissions: discord.PermissionViewChannel}, false); err != nil {
		t.Fatal(err)
	}
	if err := state.Cabinet.MemberSet(testGuildID, &discord.Member{User: discord.User{ID: testUserID}}, false); err != nil {
		t.Fatal(err)
	}
	for _, channel := range channels {
		if err := state.Cabinet.ChannelSet(&channel, false); err != nil {
			t.Fatal(err)
		}
	}

	return state
}

func channelNode(t *testing.T, m Model, id discord.ChannelID) *tree.Node {
	t.Helper()

	node := m.nodes[discord.Snowflake(id)]
	if node == nil {
		t.Fatalf("nodes[%d] = %v, want a node", id, node)
	}

	return node
}
