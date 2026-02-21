package chat

import (
	"context"
	"log/slog"
	"slices"

	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/discordo/internal/notifications"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/state/store/defaultstore"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
)

func (v *View) OpenState(token string) error {
	identifyProps := http.IdentifyProperties()
	gateway.DefaultIdentity = identifyProps
	gateway.DefaultPresence = &gateway.UpdatePresenceCommand{
		Status: v.cfg.Status,
	}

	id := gateway.DefaultIdentifier(token)
	id.Compress = false

	session := session.NewCustom(id, http.NewClient(token), handler.New())
	state := state.NewFromSession(session, defaultstore.New())
	v.state = ningen.FromState(state)

	// Handlers
	v.state.AddHandler(v.onRaw)
	v.state.AddHandler(v.onReady)
	v.state.AddHandler(v.onMessageCreate)
	v.state.AddHandler(v.onMessageUpdate)
	v.state.AddHandler(v.onMessageDelete)
	v.state.AddHandler(v.onReadUpdate)
	v.state.AddHandler(v.onGuildMembersChunk)
	v.state.AddHandler(v.onGuildMemberRemove)

	if v.cfg.TypingIndicator.Receive {
		v.state.AddHandler(v.onTypingStart)
	}

	v.state.StateLog = func(err error) {
		slog.Error("state log", "err", err)
	}

	v.state.OnRequest = append(v.state.OnRequest, httputil.WithHeaders(http.Headers()), v.onRequest)
	return v.state.Open(context.TODO())
}

func (v *View) CloseState() error {
	if v.state == nil {
		return nil
	}
	return v.state.Close()
}

func (v *View) onRequest(r httpdriver.Request) error {
	if req, ok := r.(*httpdriver.DefaultRequest); ok {
		slog.Debug("new HTTP request", "method", req.Method, "url", req.URL)
	}

	return nil
}

func (v *View) onRaw(event *ws.RawEvent) {
	slog.Debug(
		"new raw event",
		"code", event.OriginalCode,
		"type", event.OriginalType,
		// "data", event.Raw,
	)
}

func (v *View) onReady(event *gateway.ReadyEvent) {
	// Rebuild indexes from scratch so reconnects and account switches do not
	// retain stale pointers to detached tree nodes.
	v.guildsTree.resetNodeIndex()

	dmNode := tview.NewTreeNode("Direct Messages").
		SetReference(dmNode{}).
		Collapse()
	v.guildsTree.dmRootNode = dmNode

	root := v.guildsTree.
		GetRoot().
		ClearChildren().
		AddChild(dmNode)

	// Track guilds already in folders to find orphans (newly joined guilds may not be synced to GuildFolders yet but always appear in GuildPositions)
	guildsInFolders := make(map[discord.GuildID]bool)
	for _, folder := range event.UserSettings.GuildFolders {
		for _, guildID := range folder.GuildIDs {
			guildsInFolders[guildID] = true
		}
	}

	// Build index of all available guilds.
	guildsByID := make(map[discord.GuildID]*gateway.GuildCreateEvent, len(event.Guilds))
	for index := range event.Guilds {
		guildsByID[event.Guilds[index].ID] = &event.Guilds[index]
	}

	// Use GuildPositions for ordering (it's the canonical order).
	// Guilds not in any folder are "orphans" - add them directly to root.
	positions := event.UserSettings.GuildPositions
	// Fallback: GuildPositions shouldn't be nil but handle gracefully
	if len(positions) == 0 {
		positions = make([]discord.GuildID, 0, len(event.Guilds))
		for _, guildEvent := range event.Guilds {
			positions = append(positions, guildEvent.ID)
		}
	}

	for _, guildID := range positions {
		// Already handled in folder processing below
		if guildsInFolders[guildID] {
			continue
		}

		// Orphan guild - add directly to root in order
		if guildEvent, ok := guildsByID[guildID]; ok {
			v.guildsTree.createGuildNode(root, guildEvent.Guild)
		}
	}

	// Process folders (real folders and single-guild "folders")
	for _, folder := range event.UserSettings.GuildFolders {
		if folder.ID == 0 && len(folder.GuildIDs) == 1 {
			if guild, ok := guildsByID[folder.GuildIDs[0]]; ok {
				v.guildsTree.createGuildNode(root, guild.Guild)
			}
		} else {
			v.guildsTree.createFolderNode(folder, guildsByID)
		}
	}

	v.guildsTree.SetCurrentNode(dmNode)
	v.app.SetFocus(v.guildsTree)
	v.app.Draw()
}

func (v *View) onMessageCreate(message *gateway.MessageCreateEvent) {
	selectedChannel := v.SelectedChannel()
	if selectedChannel != nil && selectedChannel.ID == message.ChannelID {
		v.removeTyper(message.Author.ID)
		v.app.QueueUpdateDraw(func() {
			v.messagesList.addMessage(message.Message)
		})
	} else {
		if err := notifications.Notify(v.state, message, v.cfg); err != nil {
			slog.Error("failed to notify", "err", err, "channel_id", message.ChannelID, "message_id", message.ID)
		}
	}
}

func (v *View) onMessageUpdate(message *gateway.MessageUpdateEvent) {
	if selected := v.SelectedChannel(); selected != nil && selected.ID == message.ChannelID {
		index := slices.IndexFunc(v.messagesList.messages, func(m discord.Message) bool {
			return m.ID == message.ID
		})
		if index < 0 {
			return
		}

		v.app.QueueUpdateDraw(func() {
			v.messagesList.setMessage(index, message.Message)
		})
	}
}

func (v *View) onMessageDelete(message *gateway.MessageDeleteEvent) {
	if selected := v.SelectedChannel(); selected != nil && selected.ID == message.ChannelID {
		prevCursor := v.messagesList.Cursor()
		deletedIndex := slices.IndexFunc(v.messagesList.messages, func(m discord.Message) bool {
			return m.ID == message.ID
		})
		if deletedIndex < 0 {
			return
		}

		v.app.QueueUpdateDraw(func() {
			v.messagesList.deleteMessage(deletedIndex)
		})

		// Keep cursor stable when possible after removal.
		newCursor := prevCursor
		if prevCursor == deletedIndex {
			// Prefer previous item; fall forward if we deleted the first.
			newCursor = deletedIndex - 1
			if newCursor < 0 {
				if deletedIndex < len(v.messagesList.messages) {
					newCursor = deletedIndex
				} else {
					newCursor = -1
				}
			}
		} else if prevCursor > deletedIndex {
			// Shift back since the list shrank before the cursor.
			newCursor = prevCursor - 1
		}
		if newCursor != prevCursor {
			// Avoid redundant cursor updates if nothing changed.
			v.app.QueueUpdateDraw(func() {
				v.messagesList.SetCursor(newCursor)
			})
		}
	}
}

func (v *View) onGuildMembersChunk(event *gateway.GuildMembersChunkEvent) {
	v.messagesList.setFetchingChunk(false, uint(len(event.Members)))
}

func (v *View) onGuildMemberRemove(event *gateway.GuildMemberRemoveEvent) {
	v.messageInput.cache.Invalidate(event.GuildID.String()+" "+event.User.Username, v.state.MemberState.SearchLimit)
}

func (v *View) onTypingStart(event *gateway.TypingStartEvent) {
	selectedChannel := v.SelectedChannel()
	if selectedChannel == nil {
		return
	}

	if selectedChannel.ID != event.ChannelID {
		return
	}

	me, _ := v.state.Cabinet.Me()
	if event.UserID == me.ID {
		return
	}

	v.addTyper(event.UserID)
}
