package memberstree

import (
	"github.com/ayn2op/ningen/v3/states/member"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/tree"
)

// Msg moves or scrolls the tree.
type Msg tree.Change

// requestSelectedMembers requests the chunk of the member list after the selected member, so that the members below it load before they are reached.
func (m *Model) requestSelectedMembers() tview.Cmd {
	node := m.selectionState.CurrentNode()
	if node == nil {
		return nil
	}
	ref, ok := node.Reference().(memberRef)
	if !ok || ref.index < 0 {
		return nil
	}
	return m.requestMembers(member.ChunkFromIndex(ref.index) + 1)
}

// requestMembers requests the member list of the selected guild channel up to chunk.
func (m *Model) requestMembers(chunk int) tview.Cmd {
	channel := m.channel
	if !m.Shown() || !channel.GuildID.IsValid() {
		return nil
	}
	state := m.state
	return func() tview.Msg {
		state.MemberState.RequestMemberList(channel.GuildID, channel.ID, chunk)
		return nil
	}
}
