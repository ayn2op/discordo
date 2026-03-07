package chat

import (
	"slices"
	"testing"

	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	arikawastate "github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v3"
)

func newGuildTreeTestView(t *testing.T) *View {
	t.Helper()

	view := NewView(tview.NewApplication(), testConfig(t), "")
	view.state = ningen.FromState(arikawastate.New(""))
	view.guildsTree.resetNodeIndex()
	view.guildsTree.GetRoot().ClearChildren()
	return view
}

func addVoiceChannelNode(t *testing.T, view *View, channel discord.Channel) *tview.TreeNode {
	t.Helper()

	if err := view.state.Cabinet.ChannelSet(&channel, true); err != nil {
		t.Fatalf("set channel: %v", err)
	}

	node := tview.NewTreeNode(channel.Name).SetReference(channel.ID)
	view.guildsTree.channelNodeByID[channel.ID] = node
	view.guildsTree.GetRoot().AddChild(node)
	return node
}

func addVoiceState(t *testing.T, view *View, guildID discord.GuildID, voiceState discord.VoiceState) {
	t.Helper()

	if err := view.state.Cabinet.VoiceStateSet(guildID, &voiceState, true); err != nil {
		t.Fatalf("set voice state: %v", err)
	}
}

func addMember(t *testing.T, view *View, guildID discord.GuildID, member discord.Member) {
	t.Helper()

	if err := view.state.Cabinet.MemberSet(guildID, &member, true); err != nil {
		t.Fatalf("set member: %v", err)
	}
}

func TestGuildsTreeVoiceVisibilityCommands(t *testing.T) {
	view := newGuildTreeTestView(t)
	gt := view.guildsTree

	guildID := discord.GuildID(1)
	voiceOne := addVoiceChannelNode(t, view, discord.Channel{
		ID:      discord.ChannelID(10),
		GuildID: guildID,
		Name:    "Lobby",
		Type:    discord.GuildVoice,
	})
	addVoiceChannelNode(t, view, discord.Channel{
		ID:      discord.ChannelID(11),
		GuildID: guildID,
		Name:    "Stage",
		Type:    discord.GuildStageVoice,
	})

	addVoiceState(t, view, guildID, discord.VoiceState{
		UserID:    discord.UserID(20),
		ChannelID: discord.ChannelID(10),
		Member: &discord.Member{
			User: discord.User{ID: discord.UserID(20), Username: "bravo"},
			Nick: "Bravo",
		},
	})
	addMember(t, view, guildID, discord.Member{
		User: discord.User{ID: discord.UserID(21), Username: "alpha"},
	})
	addVoiceState(t, view, guildID, discord.VoiceState{
		UserID:    discord.UserID(21),
		ChannelID: discord.ChannelID(10),
	})
	addVoiceState(t, view, guildID, discord.VoiceState{
		UserID:    discord.UserID(22),
		ChannelID: discord.ChannelID(11),
		Member: &discord.Member{
			User: discord.User{ID: discord.UserID(22), Username: "charlie"},
		},
	})

	gt.SetCurrentNode(voiceOne)
	gt.showCurrentVoiceUsers()

	assertVoiceUsers(t, voiceUserLabels(gt, discord.ChannelID(10)), "Bravo", "alpha")
	assertVoiceUsers(t, voiceUserLabels(gt, discord.ChannelID(11)))

	participantNode := voiceOne.GetChildren()[0]
	gt.SetCurrentNode(participantNode)
	gt.hideCurrentVoiceUsers()
	assertVoiceUsers(t, voiceUserLabels(gt, discord.ChannelID(10)))

	gt.setAllVoiceUsersVisibility(true)
	assertVoiceUsers(t, voiceUserLabels(gt, discord.ChannelID(10)), "Bravo", "alpha")
	assertVoiceUsers(t, voiceUserLabels(gt, discord.ChannelID(11)), "charlie")
	if len(gt.voiceChannelVisibility) != 0 {
		t.Fatalf("expected global show-all to clear per-channel overrides, got %#v", gt.voiceChannelVisibility)
	}

	gt.setAllVoiceUsersVisibility(false)
	assertVoiceUsers(t, voiceUserLabels(gt, discord.ChannelID(10)))
	assertVoiceUsers(t, voiceUserLabels(gt, discord.ChannelID(11)))
}

func TestGuildsTreeVoiceParticipantSelectionIsNoOp(t *testing.T) {
	view := newGuildTreeTestView(t)
	gt := view.guildsTree

	guildID := discord.GuildID(2)
	voiceNode := addVoiceChannelNode(t, view, discord.Channel{
		ID:      discord.ChannelID(30),
		GuildID: guildID,
		Name:    "Lobby",
		Type:    discord.GuildVoice,
	})
	addVoiceState(t, view, guildID, discord.VoiceState{
		UserID:    discord.UserID(40),
		ChannelID: discord.ChannelID(30),
		Member: &discord.Member{
			User: discord.User{ID: discord.UserID(40), Username: "delta"},
		},
	})

	textChannel := &discord.Channel{ID: discord.ChannelID(99), Name: "general", Type: discord.GuildText}
	view.SetSelectedChannel(textChannel)

	gt.SetCurrentNode(voiceNode)
	gt.showCurrentVoiceUsers()
	participantNode := voiceNode.GetChildren()[0]

	gt.onSelected(participantNode)

	if got := view.SelectedChannel(); got != textChannel {
		t.Fatalf("participant selection changed selected channel to %+v", got)
	}
	if view.voicePanel.shouldShow() {
		t.Fatal("participant selection unexpectedly changed voice panel state")
	}
}

func TestGuildsTreeVoiceUsersRefreshAndPersistAcrossRebuild(t *testing.T) {
	view := newGuildTreeTestView(t)
	gt := view.guildsTree

	guildID := discord.GuildID(3)
	channelID := discord.ChannelID(50)
	voiceNode := addVoiceChannelNode(t, view, discord.Channel{
		ID:      channelID,
		GuildID: guildID,
		Name:    "Lounge",
		Type:    discord.GuildVoice,
	})

	addVoiceState(t, view, guildID, discord.VoiceState{
		UserID:    discord.UserID(60),
		ChannelID: channelID,
		Member: &discord.Member{
			User: discord.User{ID: discord.UserID(60), Username: "echo"},
		},
	})

	gt.SetCurrentNode(voiceNode)
	gt.showCurrentVoiceUsers()

	addVoiceState(t, view, guildID, discord.VoiceState{
		UserID:    discord.UserID(61),
		ChannelID: channelID,
		Member: &discord.Member{
			User: discord.User{ID: discord.UserID(61), Username: "foxtrot"},
		},
	})
	gt.refreshVoiceChannelUsersForGuild(guildID)

	assertVoiceUsers(t, voiceUserLabels(gt, channelID), "echo", "foxtrot")

	gt.resetNodeIndex()
	gt.GetRoot().ClearChildren()

	rebuiltNode := addVoiceChannelNode(t, view, discord.Channel{
		ID:      channelID,
		GuildID: guildID,
		Name:    "Lounge",
		Type:    discord.GuildVoice,
	})
	gt.syncVoiceChannelUsers(guildID, channelID)

	if rebuiltNode != gt.findNodeByReference(channelID) {
		t.Fatal("expected rebuilt voice node to be indexed")
	}
	assertVoiceUsers(t, voiceUserLabels(gt, channelID), "echo", "foxtrot")
}

func TestGuildsTreeVoiceVisibilityKeybinds(t *testing.T) {
	view := newGuildTreeTestView(t)
	gt := view.guildsTree

	guildID := discord.GuildID(4)
	channelID := discord.ChannelID(70)
	voiceNode := addVoiceChannelNode(t, view, discord.Channel{
		ID:      channelID,
		GuildID: guildID,
		Name:    "Call",
		Type:    discord.GuildVoice,
	})
	addVoiceState(t, view, guildID, discord.VoiceState{
		UserID:    discord.UserID(80),
		ChannelID: channelID,
		Member: &discord.Member{
			User: discord.User{ID: discord.UserID(80), Username: "golf"},
		},
	})

	gt.SetCurrentNode(voiceNode)
	gt.HandleEvent(tcell.NewEventKey(tcell.KeyRune, "v", tcell.ModNone))
	assertVoiceUsers(t, voiceUserLabels(gt, channelID), "golf")

	gt.HandleEvent(tcell.NewEventKey(tcell.KeyRune, "A", tcell.ModShift))
	assertVoiceUsers(t, voiceUserLabels(gt, channelID))
}

func TestGuildsTreeFolderNodeStillExpands(t *testing.T) {
	view := newGuildTreeTestView(t)
	view.cfg.Theme.GuildsTree.AutoExpandFolders = false

	guildID := discord.GuildID(90)
	view.guildsTree.createFolderNode(gateway.GuildFolder{
		Name:     "Folder",
		GuildIDs: []discord.GuildID{guildID},
	}, map[discord.GuildID]*gateway.GuildCreateEvent{
		guildID: {
			Guild: discord.Guild{
				ID:   guildID,
				Name: "Guild",
			},
		},
	})

	folderNode := view.guildsTree.GetRoot().GetChildren()[0]
	if folderNode.IsExpanded() {
		t.Fatal("expected folder node to start collapsed")
	}

	view.guildsTree.onSelected(folderNode)

	if !folderNode.IsExpanded() {
		t.Fatal("expected folder node to expand on selection")
	}
}

func voiceUserLabels(tree *guildsTree, channelID discord.ChannelID) []string {
	node := tree.findNodeByReference(channelID)
	if node == nil {
		return nil
	}

	labels := make([]string, 0, len(node.GetChildren()))
	for _, child := range node.GetChildren() {
		if _, ok := child.GetReference().(voiceUserNode); !ok {
			continue
		}

		var label string
		for _, segment := range child.GetLine() {
			label += segment.Text
		}
		labels = append(labels, label)
	}

	slices.Sort(labels)
	return labels
}

func assertVoiceUsers(t *testing.T, got []string, want ...string) {
	t.Helper()

	if len(got) != len(want) {
		t.Fatalf("voice users mismatch. got=%#v want=%#v", got, want)
	}

	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("voice users mismatch at index %d. got=%#v want=%#v", i, got, want)
		}
	}
}
