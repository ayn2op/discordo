package chat

import (
	"context"
	"log/slog"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
)

func openState(state *ningen.State) tview.Cmd {
	return func() tview.Msg {
		if err := state.Open(context.Background()); err != nil {
			slog.Error("failed to open chat state", "err", err)
			return nil
		}
		return nil
	}
}

func closeState(state *ningen.State) tview.Cmd {
	if state == nil {
		return nil
	}
	return func() tview.Msg {
		if err := state.Close(); err != nil {
			slog.Error("failed to close the session", "err", err)
		}
		return nil
	}
}

func listen(events <-chan gateway.Event) tview.Cmd {
	return func() tview.Msg {
		return <-events
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

type deleteMessageMsg discord.Message

type LogoutMsg struct{}

func logout() tview.Cmd {
	return func() tview.Msg {
		return LogoutMsg{}
	}
}

type QuitMsg struct{}
