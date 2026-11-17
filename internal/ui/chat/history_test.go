package chat

import (
	"slices"
	"testing"

	"github.com/ayn2op/arikawa/v3/discord"
)

// visited returns a history that opened channels in the order given.
func visited(channelIDs ...discord.ChannelID) history {
	h := newHistory()
	for _, channelID := range channelIDs {
		h.visit(channelID)
	}

	return h
}

func TestHistoryVisit(t *testing.T) {
	t.Run("keeps the channels in the order they were opened", func(t *testing.T) {
		h := visited(1, 2, 3)
		if want := []discord.ChannelID{1, 2, 3}; !slices.Equal(h.channels, want) {
			t.Fatalf("channels = %v, want %v", h.channels, want)
		}
		if h.open != 2 {
			t.Fatalf("open = %d, want 2", h.open)
		}
	})

	t.Run("opening the open channel again adds nothing", func(t *testing.T) {
		h := visited(1, 2, 2)
		if want := []discord.ChannelID{1, 2}; !slices.Equal(h.channels, want) {
			t.Fatalf("channels = %v, want %v", h.channels, want)
		}
	})

	t.Run("opening a channel from behind leaves what was ahead", func(t *testing.T) {
		h := visited(1, 2, 3)
		h.move(stepBack)
		h.move(stepBack)

		h.visit(4)
		if want := []discord.ChannelID{1, 4}; !slices.Equal(h.channels, want) {
			t.Fatalf("channels = %v, want %v", h.channels, want)
		}
		if h.open != 1 {
			t.Fatalf("open = %d, want 1", h.open)
		}
	})

	t.Run("a channel opened twice over is in the path twice", func(t *testing.T) {
		h := visited(1, 2, 1)
		if want := []discord.ChannelID{1, 2, 1}; !slices.Equal(h.channels, want) {
			t.Fatalf("channels = %v, want %v", h.channels, want)
		}
	})
}

func TestHistoryAt(t *testing.T) {
	for _, test := range []struct {
		name    string
		history history
		step    int
		want    discord.ChannelID
		wantOk  bool
	}{
		{"the open channel", visited(1, 2), 0, 2, true},
		{"the one before it", visited(1, 2), stepBack, 1, true},
		{"nothing before the first", visited(1), stepBack, discord.NullChannelID, false},
		{"nothing after the last", visited(1), stepForward, discord.NullChannelID, false},
		{"nothing at all before a channel is opened", newHistory(), 0, discord.NullChannelID, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, ok := test.history.at(test.step)
			if ok != test.wantOk {
				t.Fatalf("ok = %v, want %v", ok, test.wantOk)
			}
			if got != test.want {
				t.Fatalf("got = %v, want %v", got, test.want)
			}
		})
	}
}

func TestHistoryMove(t *testing.T) {
	t.Run("walks back and forward over the same path", func(t *testing.T) {
		h := visited(1, 2, 3)
		h.move(stepBack)
		if got, _ := h.at(0); got != 2 {
			t.Fatalf("after going back, the open channel = %v, want 2", got)
		}

		h.move(stepForward)
		if got, _ := h.at(0); got != 3 {
			t.Fatalf("after going forward, the open channel = %v, want 3", got)
		}
		if want := []discord.ChannelID{1, 2, 3}; !slices.Equal(h.channels, want) {
			t.Fatalf("channels = %v, want %v", h.channels, want)
		}
	})
}

func TestModelOpenInHistory(t *testing.T) {
	t.Run("records every channel opened", func(t *testing.T) {
		m := newTestModel(t)
		m.setSelectedChannel(&discord.Channel{ID: 1})
		m.setSelectedChannel(&discord.Channel{ID: 2})

		if want := []discord.ChannelID{1, 2}; !slices.Equal(m.history.channels, want) {
			t.Fatalf("channels = %v, want %v", m.history.channels, want)
		}
	})

	t.Run("goes nowhere from the only channel opened", func(t *testing.T) {
		m := newTestModel(t)
		m.setSelectedChannel(&discord.Channel{ID: 1})

		for _, step := range []int{stepBack, stepForward} {
			if cmd := m.openInHistory(step); cmd != nil {
				t.Fatalf("step %d went somewhere from the only channel opened", step)
			}
			if got, _ := m.history.at(0); got != 1 {
				t.Fatalf("the open channel = %v, want 1", got)
			}
		}
	})

	t.Run("leaves the path alone when the channel has no node to open", func(t *testing.T) {
		m := newTestModel(t)
		m.setSelectedChannel(&discord.Channel{ID: 1})
		m.setSelectedChannel(&discord.Channel{ID: 2})

		// Neither channel is in the guilds tree, which has no guild expanded.
		if cmd := m.openInHistory(stepBack); cmd != nil {
			t.Fatal("a channel outside the tree was opened")
		}
		if got, _ := m.history.at(0); got != 2 {
			t.Fatalf("the open channel = %v, want 2, as the step did not happen", got)
		}
	})
}
