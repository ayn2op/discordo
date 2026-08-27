package chat

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview/tree"
)

func TestThreadListSyncClearsEmptyScope(t *testing.T) {
	const (
		guildID  discord.GuildID   = 1
		parentID discord.ChannelID = 2
		threadID discord.ChannelID = 3
	)

	state := ningen.New("")
	if err := state.Cabinet.ChannelSet(&discord.Channel{
		ID:       threadID,
		GuildID:  guildID,
		ParentID: parentID,
		Type:     discord.GuildPublicThread,
	}, false); err != nil {
		t.Fatal(err)
	}

	root := tree.NewNode("")
	parent := tree.NewNode("").SetReference(parentID)
	threadNode := tree.NewNode("").SetReference(threadID)
	root.AddChild(parent)
	parent.AddChild(threadNode)
	treeModel := tree.NewModel().SetRoot(root)
	m := &Model{
		state: state,
		guildsTree: &guildsTree{
			Model: treeModel,
			state: state,
			channelNodeByID: map[discord.ChannelID]*tree.Node{
				parentID: parent,
				threadID: threadNode,
			},
		},
	}

	m.onThreadListSync(&gateway.ThreadListSyncEvent{
		GuildID:    guildID,
		ChannelIDs: []discord.ChannelID{parentID},
	})

	if len(parent.Children()) != 0 || m.guildsTree.channelNodeByID[threadID] != nil {
		t.Fatal("stale thread was not removed")
	}
}
