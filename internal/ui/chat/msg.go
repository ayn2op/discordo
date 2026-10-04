package chat

import (
	"context"
	"log/slog"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/ui/chat/composer"
	"github.com/ayn2op/discordo/internal/ui/chat/guildstree"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
)

func openState(state *ningen.State) tview.Cmd {
	return func() tview.Msg {
		if err := state.Open(context.Background()); err != nil {
			slog.Error("failed to open chat state", "err", err)
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

type LogoutMsg struct{}

func logout() tview.Cmd {
	return func() tview.Msg {
		return LogoutMsg{}
	}
}

type QuitMsg struct{}

func (m *Model) setSelectedChannel(channel *discord.Channel) tview.Cmd {
	m.selectedChannel = channel
	m.composer.SetChannel(channel)
	cmd := m.membersTree.SetChannel(channel)
	if !m.canFocus(m.focused) {
		m.focused = paneMessagesList
	}
	return cmd
}

func (m *Model) navigateToChannel(channelID discord.ChannelID) tview.Cmd {
	m.closePicker()
	var cmd tview.Cmd
	m.guildsTree, cmd = m.guildsTree.Update(guildstree.NavigateMsg{ChannelID: channelID})
	return cmd
}

// paneMsg is what a pane's widget made of a mouse message within it. A left mouse button press also focuses the pane.
type paneMsg struct {
	pane  pane
	msg   tview.Msg
	focus bool
}

// typingExpiredMsg ends a user's typing indicator unless they typed again and extended it.
type typingExpiredMsg struct {
	userID discord.UserID
}

// addTyper shows userID as typing and returns a command that ends it after the typing duration.
func (m *Model) addTyper(userID discord.UserID) tview.Cmd {
	m.typers[userID] = time.Now().Add(composer.TypingDuration)
	return func() tview.Msg {
		time.Sleep(composer.TypingDuration)
		return typingExpiredMsg{userID}
	}
}
