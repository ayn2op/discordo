package chat

import (
	"context"
	"log/slog"
	"math"
	"time"

	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
)

func (m *Model) openState() tview.Cmd {
	return func() tview.Msg {
		if err := m.state.Open(context.Background()); err != nil {
			slog.Error("failed to open chat state", "err", err)
			return nil
		}
		return nil
	}
}

func (m *Model) closeState() tview.Cmd {
	if m.state == nil {
		return nil
	}
	return func() tview.Msg {
		if err := m.state.Close(); err != nil {
			slog.Error("failed to close the session", "err", err)
		}
		return nil
	}
}

func (m *Model) listen() tview.Cmd {
	return func() tview.Msg {
		return <-m.events
	}
}

type channelLoadedMsg struct {
	Channel  discord.Channel
	Messages []discord.Message
}

type olderMessagesLoadedMsg struct {
	ChannelID discord.ChannelID
	Older     []discord.Message
}

type LogoutMsg struct{}

func (m *Model) logout() tview.Cmd {
	return func() tview.Msg {
		return LogoutMsg{}
	}
}

type QuitMsg struct{}

// reconnectWithBackoff attempts to reconnect to the Discord gateway with
// exponential backoff. It retries up to maxReconnectAttempts times.
func (m *Model) reconnectWithBackoff() tview.Cmd {
	const (
		maxReconnectAttempts = 5
		baseDelay            = 2 * time.Second
		maxDelay             = 60 * time.Second
	)

	return func() tview.Msg {
		for attempt := 0; attempt < maxReconnectAttempts; attempt++ {
			if m.connected {
				return nil
			}

			slog.Info("attempting to reconnect", "attempt", attempt+1, "max_attempts", maxReconnectAttempts)
			if err := m.state.Open(context.Background()); err != nil {
				slog.Error("reconnection failed", "attempt", attempt+1, "err", err)

				delay := time.Duration(math.Min(
					float64(baseDelay)*math.Pow(2, float64(attempt)),
					float64(maxDelay),
				))
				time.Sleep(delay)
				continue
			}

			slog.Info("reconnection successful", "attempt", attempt+1)
			return nil
		}

		slog.Error("failed to reconnect after maximum attempts", "attempts", maxReconnectAttempts)
		return nil
	}
}
