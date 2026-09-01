package guildstree

import (
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/tview/tree"
)

func TestMoveDMToFront(t *testing.T) {
	firstID := discord.ChannelID(1)
	secondID := discord.ChannelID(2)
	thirdID := discord.ChannelID(3)

	first := tree.NewNode("first").SetReference(firstID)
	second := tree.NewNode("second").SetReference(secondID)
	third := tree.NewNode("third").SetReference(thirdID)
	dmRoot := tree.NewNode("Direct Messages").SetChildren([]*tree.Node{first, second, third})
	m := Model{
		dmRootNode: dmRoot,
		channelNodeByID: map[discord.ChannelID]*tree.Node{
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
}

func TestMoveDMToFrontIgnoresUnrenderedChannel(t *testing.T) {
	firstID := discord.ChannelID(1)
	first := tree.NewNode("first").SetReference(firstID)
	dmRoot := tree.NewNode("Direct Messages").SetChildren([]*tree.Node{first})
	m := Model{
		dmRootNode:      dmRoot,
		channelNodeByID: map[discord.ChannelID]*tree.Node{},
	}

	m.moveDMToFront(discord.ChannelID(2))

	children := dmRoot.Children()
	if len(children) != 1 || children[0] != first {
		t.Fatalf("children = %v, want the original DM node", children)
	}
}
