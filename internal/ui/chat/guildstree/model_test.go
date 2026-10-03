package guildstree

import (
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
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
