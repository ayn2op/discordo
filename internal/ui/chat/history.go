package chat

import (
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/tview"
)

// step is how far along the path a key moves.
const (
	stepBack    = -1
	stepForward = 1
)

// history is the path of channels opened in this session, walked back and forth as a browser walks its own.
// Opening a channel from anywhere but the path itself drops whatever lay ahead of the open one:
//
//	a → b → c    opened in turn, the path ends at c
//	a           two steps back, the path is at a, with b and c ahead
//	a → d        opening d from there leaves b and c behind
type history struct {
	channels []discord.ChannelID
	// open is where the open channel sits in channels, and is -1 until the first one is opened.
	open int
}

func newHistory() history {
	return history{open: -1}
}

// visit puts channelID at the end of the path, unless it is the open channel already.
func (h *history) visit(channelID discord.ChannelID) {
	if current, ok := h.at(0); ok && current == channelID {
		return
	}

	h.channels = append(h.channels[:h.open+1], channelID)
	h.open = len(h.channels) - 1
}

// at returns the channel step places from the open one, and whether the path reaches that far.
func (h history) at(step int) (discord.ChannelID, bool) {
	place := h.open + step
	if place < 0 || place >= len(h.channels) {
		return discord.NullChannelID, false
	}

	return h.channels[place], true
}

// move takes the open channel step places along the path.
func (h *history) move(step int) {
	h.open += step
}

// openInHistory opens the channel step places from the open one along the path walked this session.
func (m *Model) openInHistory(step int) tview.Cmd {
	channelID, ok := m.history.at(step)
	if !ok {
		return nil
	}

	// A channel of a guild never expanded has no node to open, and the path stays where it is rather
	// than counting a step that did not happen.
	cmd := m.navigateToChannel(channelID)
	if cmd == nil {
		return nil
	}

	m.history.move(step)
	return cmd
}
