package chat

import (
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/tview/tree"
)

func TestMessageCreatePromotesDMChannel(t *testing.T) {
	firstID := discord.ChannelID(1)
	secondID := discord.ChannelID(2)
	thirdID := discord.ChannelID(3)

	first := tree.NewNode("first").SetReference(firstID)
	second := tree.NewNode("second").SetReference(secondID)
	third := tree.NewNode("third").SetReference(thirdID)
	dmRoot := tree.NewNode("Direct Messages").SetChildren([]*tree.Node{first, second, third})
	m := Model{guildsTree: &guildsTree{
		dmRootNode: dmRoot,
		channelNodeByID: map[discord.ChannelID]*tree.Node{
			firstID:  first,
			secondID: second,
			thirdID:  third,
		},
	}}

	m.onMessageCreate(&gateway.MessageCreateEvent{Message: discord.Message{ChannelID: thirdID}})

	want := []*tree.Node{third, first, second}
	if !slices.Equal(dmRoot.Children(), want) {
		t.Fatalf("children = %v, want %v", dmRoot.Children(), want)
	}
}

func TestMessageCreateIgnoresUnrenderedDMChannel(t *testing.T) {
	firstID := discord.ChannelID(1)
	first := tree.NewNode("first").SetReference(firstID)
	dmRoot := tree.NewNode("Direct Messages").SetChildren([]*tree.Node{first})
	m := Model{guildsTree: &guildsTree{
		dmRootNode:      dmRoot,
		channelNodeByID: map[discord.ChannelID]*tree.Node{},
	}}

	m.onMessageCreate(&gateway.MessageCreateEvent{Message: discord.Message{ChannelID: 2}})

	children := dmRoot.Children()
	if len(children) != 1 || children[0] != first {
		t.Fatalf("children = %v, want the original DM node", children)
	}
}
