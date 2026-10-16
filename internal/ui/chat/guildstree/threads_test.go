package guildstree

import (
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview/tree"
)

func TestSyncThreads(t *testing.T) {
	const (
		guildID  discord.GuildID   = 1
		parentID discord.ChannelID = 2
		threadID discord.ChannelID = 3
	)

	newModel := func(t *testing.T) (Model, *tree.Node) {
		t.Helper()

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
		return Model{
			state: state,
			nodes: map[discord.Snowflake]*tree.Node{
				discord.Snowflake(parentID): parent,
				discord.Snowflake(threadID): threadNode,
			},
		}, parent
	}

	assertCleared := func(t *testing.T, m Model, parent *tree.Node) {
		t.Helper()

		if got := parent.Children(); len(got) != 0 {
			t.Fatalf("parent children = %v, want %v", got, []*tree.Node{})
		}
		if node := m.nodes[discord.Snowflake(threadID)]; node != nil {
			t.Fatalf("nodes[%d] = %v, want %v", threadID, node, nil)
		}
	}

	t.Run("clears the parent the event names", func(t *testing.T) {
		m, parent := newModel(t)

		m.syncThreads(&gateway.ThreadListSyncEvent{
			GuildID:    guildID,
			ChannelIDs: []discord.ChannelID{parentID},
		})

		assertCleared(t, m, parent)
	})

	t.Run("clears the whole guild when the event names no channels", func(t *testing.T) {
		m, parent := newModel(t)

		m.syncThreads(&gateway.ThreadListSyncEvent{GuildID: guildID})

		assertCleared(t, m, parent)
	})
}
