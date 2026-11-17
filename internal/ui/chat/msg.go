package chat

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/utils/ws"
	"github.com/ayn2op/discordo/internal/ui/chat/composer"
	"github.com/ayn2op/discordo/internal/ui/chat/guildstree"
	"github.com/ayn2op/discordo/internal/ui/chat/reactionspicker"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/cmd"
)

// closeCodeAuthenticationFailed is the close code of the gateway for a token it rejects.
const closeCodeAuthenticationFailed = 4004

func openState(state *ningen.State) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		err := state.Open(context.Background())
		if err == nil {
			return nil
		}

		slog.Error("failed to open chat state", "err", err)
		// A rejected token never works again, so say so over the login.
		if !isAuthenticationFailed(err) {
			return nil
		}

		return LogoutMsg{Err: err}
	})
}

func isAuthenticationFailed(err error) bool {
	event, ok := errors.AsType[*ws.CloseEvent](err)
	return ok && event.Code == closeCodeAuthenticationFailed
}

// closeState closes the session, if any, before sending next, so that next comes after the session is gone.
func closeState(state *ningen.State, next tview.Msg) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		if state != nil {
			if err := state.Close(); err != nil {
				slog.Error("failed to close the session", "err", err)
			}
		}
		return next
	})
}

func listen(events <-chan gateway.Event) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		return <-events
	})
}

// LogoutMsg asks for the login screen, with Err, if set, as why the session ended.
type LogoutMsg struct{ Err error }

type (
	QuitMsg        struct{}
	StateClosedMsg struct{}
)

func (m *Model) setSelectedChannel(channel *discord.Channel) tview.Cmd {
	m.selectedChannel = channel
	if channel != nil {
		m.history.visit(channel.ID)
	}

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
func (m Model) addTyper(userID discord.UserID) tview.Cmd {
	m.typers[userID] = time.Now().Add(composer.TypingDuration)
	return cmd.Task(func() tview.Msg {
		time.Sleep(composer.TypingDuration)
		return typingExpiredMsg{userID}
	})
}

// react reacts as msg asks. The reaction is shown once the gateway reports it.
func (m Model) react(msg reactionspicker.SelectedMsg) tview.Cmd {
	return cmd.Task(func() tview.Msg {
		if err := m.state.React(msg.ChannelID, msg.MessageID, msg.Emoji); err != nil {
			slog.Error("failed to react", "channel_id", msg.ChannelID, "message_id", msg.MessageID, "emoji", msg.Emoji, "err", err)
		}
		return nil
	})
}
