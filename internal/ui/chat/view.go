package chat

import (
	"log/slog"
	"sync"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/notifications"
	"github.com/ayn2op/discordo/internal/profile"
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

type View struct {
	*tview.Pages

	mainFlex  *tview.Flex
	rightFlex *tview.Flex

	guildsTree   *guildsTree
	messagesList *messagesList
	messageInput *messageInput

	selectedChannel   *discord.Channel
	selectedChannelMu sync.RWMutex

	app          *tview.Application
	cfg          *config.Config
	state        *ningen.State
	profileCache *profile.Cache

	onLogout func()
}

func NewView(app *tview.Application, cfg *config.Config, onLogout func()) *View {
	v := &View{
		Pages: tview.NewPages(),

		mainFlex:  tview.NewFlex(),
		rightFlex: tview.NewFlex(),

		app:      app,
		cfg:      cfg,
		onLogout: onLogout,
	}
	v.guildsTree = newGuildsTree(cfg, v)
	v.messagesList = newMessagesList(cfg, v)
	v.messageInput = newMessageInput(cfg, v)

	v.SetInputCapture(v.onInputCapture)
	v.buildLayout()
	return v
}

func (v *View) SelectedChannel() *discord.Channel {
	v.selectedChannelMu.RLock()
	defer v.selectedChannelMu.RUnlock()
	return v.selectedChannel
}

func (v *View) SetSelectedChannel(channel *discord.Channel) {
	v.selectedChannelMu.Lock()
	v.selectedChannel = channel
	v.selectedChannelMu.Unlock()
}

func (v *View) buildLayout() {
	v.Clear()
	v.rightFlex.Clear()
	v.mainFlex.Clear()

	v.rightFlex.
		SetDirection(tview.FlexRow).
		AddItem(v.messagesList, 0, 1, false).
		AddItem(v.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	v.mainFlex.
		AddItem(v.guildsTree, 0, 1, true).
		AddItem(v.rightFlex, 0, 4, false)

	v.AddAndSwitchToPage(flexPageName, v.mainFlex, true)
}

func (v *View) toggleGuildsTree() {
	// The guilds tree is visible if the number of items is two.
	if v.mainFlex.GetItemCount() == 2 {
		v.mainFlex.RemoveItem(v.guildsTree)
		if v.guildsTree.HasFocus() {
			v.app.SetFocus(v.mainFlex)
		}
	} else {
		v.buildLayout()
		v.app.SetFocus(v.guildsTree)
	}
}

func (v *View) focusGuildsTree() bool {
	// The guilds tree is not hidden if the number of items is two.
	if v.mainFlex.GetItemCount() == 2 {
		v.app.SetFocus(v.guildsTree)
		return true
	}

	return false
}

func (v *View) focusMessageInput() bool {
	if !v.messageInput.GetDisabled() {
		v.app.SetFocus(v.messageInput)
		return true
	}

	return false
}

func (v *View) focusPrevious() {
	switch v.app.GetFocus() {
	case v.guildsTree:
		v.focusMessageInput()
	case v.messagesList: // Handle both a.messagesList and a.flex as well as other edge cases (if there is).
		if ok := v.focusGuildsTree(); !ok {
			v.app.SetFocus(v.messageInput)
		}
	case v.messageInput:
		v.app.SetFocus(v.messagesList)
	}
}

func (v *View) focusNext() {
	switch v.app.GetFocus() {
	case v.guildsTree:
		v.app.SetFocus(v.messagesList)
	case v.messagesList:
		v.focusMessageInput()
	case v.messageInput: // Handle both a.messageInput and a.flex as well as other edge cases (if there is).
		if ok := v.focusGuildsTree(); !ok {
			v.app.SetFocus(v.messagesList)
		}
	}
}

func (v *View) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case v.cfg.Keys.FocusGuildsTree:
		v.messageInput.removeMentionsList()
		v.focusGuildsTree()
		return nil
	case v.cfg.Keys.FocusMessagesList:
		v.messageInput.removeMentionsList()
		v.app.SetFocus(v.messagesList)
		return nil
	case v.cfg.Keys.FocusMessageInput:
		v.focusMessageInput()
		return nil
	case v.cfg.Keys.FocusPrevious:
		v.focusPrevious()
		return nil
	case v.cfg.Keys.FocusNext:
		v.focusNext()
		return nil
	case v.cfg.Keys.Logout:
		if v.onLogout != nil {
			v.onLogout()
		}

		if err := keyring.DeleteToken(); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case v.cfg.Keys.ToggleGuildsTree:
		v.toggleGuildsTree()
		return nil
	}

	return event
}

func (v *View) showConfirmModal(prompt string, buttons []string, onDone func(label string)) {
	previousFocus := v.app.GetFocus()

	modal := tview.NewModal().
		SetText(prompt).
		AddButtons(buttons).
		SetDoneFunc(func(_ int, buttonLabel string) {
			v.RemovePage(confirmModalPageName).SwitchToPage(flexPageName)
			v.app.SetFocus(previousFocus)

			if onDone != nil {
				onDone(buttonLabel)
			}
		})

	v.
		AddAndSwitchToPage(confirmModalPageName, ui.Centered(modal, 0, 0), true).
		ShowPage(flexPageName)
}

func (v *View) onReadUpdate(event *read.UpdateEvent) {
	var guildNode *tview.TreeNode
	v.guildsTree.
		GetRoot().
		Walk(func(node, parent *tview.TreeNode) bool {
			switch node.GetReference() {
			case event.GuildID:
				node.SetTextStyle(v.guildsTree.getGuildNodeStyle(event.GuildID))
				guildNode = node
				return false
			case event.ChannelID:
				// private channel
				if !event.GuildID.IsValid() {
					style := v.guildsTree.getChannelNodeStyle(event.ChannelID)
					node.SetTextStyle(style)
					return false
				}
			}

			return true
		})

	if guildNode != nil {
		guildNode.Walk(func(node, parent *tview.TreeNode) bool {
			if node.GetReference() == event.ChannelID {
				node.SetTextStyle(v.guildsTree.getChannelNodeStyle(event.ChannelID))
				return false
			}

			return true
		})
	}

	v.app.Draw()
}

func (v *View) onReady(r *gateway.ReadyEvent) {
	dmNode := tview.NewTreeNode("Direct Messages")
	root := v.guildsTree.
		GetRoot().
		ClearChildren().
		AddChild(dmNode)

	for _, folder := range r.UserSettings.GuildFolders {
		if folder.ID == 0 && len(folder.GuildIDs) == 1 {
			guild, err := v.state.Cabinet.Guild(folder.GuildIDs[0])
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

			v.guildsTree.createGuildNode(root, *guild)
		} else {
			v.guildsTree.createFolderNode(folder)
		}
	}

	v.guildsTree.SetCurrentNode(root)
	v.app.SetFocus(v.guildsTree)
	v.app.Draw()
}

func (v *View) onMessageCreate(message *gateway.MessageCreateEvent) {
	if selected := v.SelectedChannel(); selected != nil && selected.ID == message.ChannelID {
		v.messagesList.drawMessage(v.messagesList, message.Message)
		v.messagesList.fetchProfiles([]discord.Message{message.Message})
		v.app.Draw()
	}

	if err := notifications.Notify(v.state, message, v.cfg); err != nil {
		slog.Error("failed to notify", "err", err, "channel_id", message.ChannelID, "message_id", message.ID)
	}
}

func (v *View) onMessageUpdate(message *gateway.MessageUpdateEvent) {
	if selected := v.SelectedChannel(); selected != nil && selected.ID == message.ChannelID {
		v.onMessageDelete(&gateway.MessageDeleteEvent{ID: message.ID, ChannelID: message.ChannelID, GuildID: message.GuildID})
	}
}

func (v *View) onMessageDelete(message *gateway.MessageDeleteEvent) {
	if selected := v.SelectedChannel(); selected != nil && selected.ID == message.ChannelID {
		messages, err := v.state.Cabinet.Messages(message.ChannelID)
		if err != nil {
			slog.Error("failed to get messages from state", "err", err, "channel_id", message.ChannelID)
			return
		}

		v.messagesList.reset()
		v.messagesList.drawMessages(messages)
		v.app.Draw()
	}
}

func (v *View) onGuildMembersChunk(event *gateway.GuildMembersChunkEvent) {
	v.messagesList.setFetchingChunk(false, uint(len(event.Members)))
}

func (v *View) onGuildMemberRemove(event *gateway.GuildMemberRemoveEvent) {
	v.messageInput.cache.Invalidate(event.GuildID.String()+" "+event.User.Username, v.state.MemberState.SearchLimit)
}
