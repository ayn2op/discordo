package chat

import (
	"context"
	"log/slog"

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
