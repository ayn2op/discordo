package chat

import (
	"log/slog"
	"sync"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/notifications"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/ningen/v3"
	"github.com/diamondburned/ningen/v3/states/read"
	"github.com/gdamore/tcell/v3"
)

const (
	flexPageName            = "flex"
	mentionsListPageName    = "mentionsList"
	attachmentsListPageName = "attachmentsList"
	confirmModalPageName    = "confirmModal"
)

type ChatView struct {
	*tview.Pages

	mainFlex  *tview.Flex
	rightFlex *tview.Flex

	guildsTree   *guildsTree
	messagesList *messagesList
	messageInput *messageInput

	selectedChannel   *discord.Channel
	selectedChannelMu sync.RWMutex

	app   *tview.Application
	cfg   *config.Config
	state *ningen.State

	onLogout func()
}

func NewChatView(app *tview.Application, cfg *config.Config, onLogout func()) *ChatView {
	chatView := &ChatView{
		Pages: tview.NewPages(),

		mainFlex:  tview.NewFlex(),
		rightFlex: tview.NewFlex(),

		app:      app,
		cfg:      cfg,
		onLogout: onLogout,
	}
	chatView.guildsTree = newGuildsTree(cfg, chatView)
	chatView.messagesList = newMessagesList(cfg, chatView)
	chatView.messageInput = newMessageInput(cfg, chatView)

	chatView.SetInputCapture(chatView.onInputCapture)
	chatView.buildLayout()
	return chatView
}

func (cv *ChatView) SelectedChannel() *discord.Channel {
	cv.selectedChannelMu.RLock()
	defer cv.selectedChannelMu.RUnlock()
	return cv.selectedChannel
}

func (cv *ChatView) SetSelectedChannel(channel *discord.Channel) {
	cv.selectedChannelMu.Lock()
	cv.selectedChannel = channel
	cv.selectedChannelMu.Unlock()
}

func (cv *ChatView) buildLayout() {
	cv.Clear()
	cv.rightFlex.Clear()
	cv.mainFlex.Clear()

	cv.rightFlex.
		SetDirection(tview.FlexRow).
		AddItem(cv.messagesList, 0, 1, false).
		AddItem(cv.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	cv.mainFlex.
		AddItem(cv.guildsTree, 0, 1, true).
		AddItem(cv.rightFlex, 0, 4, false)

	cv.AddAndSwitchToPage(flexPageName, cv.mainFlex, true)
}

func (cv *ChatView) toggleGuildsTree() {
	// The guilds tree is visible if the number of items is two.
	if cv.mainFlex.GetItemCount() == 2 {
		cv.mainFlex.RemoveItem(cv.guildsTree)
		if cv.guildsTree.HasFocus() {
			cv.app.SetFocus(cv.mainFlex)
		}
	} else {
		cv.buildLayout()
		cv.app.SetFocus(cv.guildsTree)
	}
}

func (cv *ChatView) focusGuildsTree() bool {
	// The guilds tree is not hidden if the number of items is two.
	if cv.mainFlex.GetItemCount() == 2 {
		cv.app.SetFocus(cv.guildsTree)
		return true
	}

	return false
}

func (cv *ChatView) focusMessageInput() bool {
	if !cv.messageInput.GetDisabled() {
		cv.app.SetFocus(cv.messageInput)
		return true
	}

	return false
}

func (cv *ChatView) focusPrevious() {
	switch cv.app.GetFocus() {
	case cv.guildsTree:
		cv.focusMessageInput()
	case cv.messagesList: // Handle both a.messagesList and a.flex as well as other edge cases (if there is).
		if ok := cv.focusGuildsTree(); !ok {
			cv.app.SetFocus(cv.messageInput)
		}
	case cv.messageInput:
		cv.app.SetFocus(cv.messagesList)
	}
}

func (cv *ChatView) focusNext() {
	switch cv.app.GetFocus() {
	case cv.guildsTree:
		cv.app.SetFocus(cv.messagesList)
	case cv.messagesList:
		cv.focusMessageInput()
	case cv.messageInput: // Handle both a.messageInput and a.flex as well as other edge cases (if there is).
		if ok := cv.focusGuildsTree(); !ok {
			cv.app.SetFocus(cv.messagesList)
		}
	}
}

func (cv *ChatView) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cv.cfg.Keys.FocusGuildsTree:
		cv.messageInput.removeMentionsList()
		cv.focusGuildsTree()
		return nil
	case cv.cfg.Keys.FocusMessagesList:
		cv.messageInput.removeMentionsList()
		cv.app.SetFocus(cv.messagesList)
		return nil
	case cv.cfg.Keys.FocusMessageInput:
		cv.focusMessageInput()
		return nil
	case cv.cfg.Keys.FocusPrevious:
		cv.focusPrevious()
		return nil
	case cv.cfg.Keys.FocusNext:
		cv.focusNext()
		return nil
	case cv.cfg.Keys.Logout:
		if cv.onLogout != nil {
			cv.onLogout()
		}

		if err := keyring.DeleteToken(); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case cv.cfg.Keys.ToggleGuildsTree:
		cv.toggleGuildsTree()
		return nil
	}

	return event
}

func (cv *ChatView) showConfirmModal(prompt string, buttons []string, onDone func(label string)) {
	previousFocus := cv.app.GetFocus()

	modal := tview.NewModal().
		SetText(prompt).
		AddButtons(buttons).
		SetDoneFunc(func(_ int, buttonLabel string) {
			cv.RemovePage(confirmModalPageName).SwitchToPage(flexPageName)
			cv.app.SetFocus(previousFocus)

			if onDone != nil {
				onDone(buttonLabel)
			}
		})

	cv.
		AddAndSwitchToPage(confirmModalPageName, ui.Centered(modal, 0, 0), true).
		ShowPage(flexPageName)
}

func (cv *ChatView) onReadUpdate(event *read.UpdateEvent) {
	var guildNode *tview.TreeNode
	cv.guildsTree.
		GetRoot().
		Walk(func(node, parent *tview.TreeNode) bool {
			switch node.GetReference() {
			case event.GuildID:
				node.SetTextStyle(cv.guildsTree.getGuildNodeStyle(event.GuildID))
				guildNode = node
				return false
			case event.ChannelID:
				// private channel
				if !event.GuildID.IsValid() {
					style := cv.guildsTree.getChannelNodeStyle(event.ChannelID)
					node.SetTextStyle(style)
					return false
				}
			}

			return true
		})

	if guildNode != nil {
		guildNode.Walk(func(node, parent *tview.TreeNode) bool {
			if node.GetReference() == event.ChannelID {
				node.SetTextStyle(cv.guildsTree.getChannelNodeStyle(event.ChannelID))
				return false
			}

			return true
		})
	}

	cv.app.Draw()
}

func (cv *ChatView) onReady(r *gateway.ReadyEvent) {
	dmNode := tview.NewTreeNode("Direct Messages")
	root := cv.guildsTree.
		GetRoot().
		ClearChildren().
		AddChild(dmNode)

	for _, folder := range r.UserSettings.GuildFolders {
		if folder.ID == 0 && len(folder.GuildIDs) == 1 {
			guild, err := cv.state.Cabinet.Guild(folder.GuildIDs[0])
			if err != nil {
				slog.Error(
					"failed to get guild from state",
					"guild_id",
					folder.GuildIDs[0],
					"err",
					err,
				)
				continue
			}

			cv.guildsTree.createGuildNode(root, *guild)
		} else {
			cv.guildsTree.createFolderNode(folder)
		}
	}

	cv.guildsTree.SetCurrentNode(root)
	cv.app.SetFocus(cv.guildsTree)
	cv.app.Draw()
}

func (cv *ChatView) onMessageCreate(message *gateway.MessageCreateEvent) {
	if selected := cv.SelectedChannel(); selected != nil && selected.ID == message.ChannelID {
		cv.messagesList.drawMessage(cv.messagesList, message.Message)
		cv.app.Draw()
	}

	if err := notifications.Notify(cv.state, message, cv.cfg); err != nil {
		slog.Error("failed to notify", "err", err, "channel_id", message.ChannelID, "message_id", message.ID)
	}
}

func (cv *ChatView) onMessageUpdate(message *gateway.MessageUpdateEvent) {
	if selected := cv.SelectedChannel(); selected != nil && selected.ID == message.ChannelID {
		cv.onMessageDelete(&gateway.MessageDeleteEvent{ID: message.ID, ChannelID: message.ChannelID, GuildID: message.GuildID})
	}
}

func (cv *ChatView) onMessageDelete(message *gateway.MessageDeleteEvent) {
	if selected := cv.SelectedChannel(); selected != nil && selected.ID == message.ChannelID {
		messages, err := cv.state.Cabinet.Messages(message.ChannelID)
		if err != nil {
			slog.Error("failed to get messages from state", "err", err, "channel_id", message.ChannelID)
			return
		}

		cv.messagesList.reset()
		cv.messagesList.drawMessages(messages)
		cv.app.Draw()
	}
}

func (cv *ChatView) onGuildMembersChunk(event *gateway.GuildMembersChunkEvent) {
	cv.messagesList.setFetchingChunk(false, uint(len(event.Members)))
}

func (cv *ChatView) onGuildMemberRemove(event *gateway.GuildMemberRemoveEvent) {
	cv.messageInput.cache.Invalidate(event.GuildID.String()+" "+event.User.Username, cv.state.MemberState.SearchLimit)
}
