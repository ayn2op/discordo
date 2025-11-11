package cmd

import (
	"context"
	"log/slog"

	"github.com/ayn2op/discordo/internal/http"
	"github.com/ayn2op/discordo/internal/notifications"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/state/store/defaultstore"
	"github.com/diamondburned/arikawa/v3/utils/handler"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
	"github.com/diamondburned/ningen/v3/states/read"
)

func openState(token string) error {
	identifyProps := http.IdentifyProperties()

	api.UserAgent = http.BrowserUserAgent
	gateway.DefaultIdentity = identifyProps
	gateway.DefaultPresence = &gateway.UpdatePresenceCommand{
		Status: app.cfg.Status,
	}

	id := gateway.DefaultIdentifier(token)
	id.Compress = false

	session := session.NewCustom(id, http.NewClient(token), handler.New())
	state := state.NewFromSession(session, defaultstore.New())
	discordState = ningen.FromState(state)

	// Handlers
	discordState.AddHandler(onRaw)
	discordState.AddHandler(onReady)
	discordState.AddHandler(onMessageCreate)
	discordState.AddHandler(onMessageUpdate)
	discordState.AddHandler(onMessageDelete)
	discordState.AddHandler(onReadUpdate)

	discordState.AddHandler(func(event *gateway.GuildMembersChunkEvent) {
		app.messagesList.setFetchingChunk(false, uint(len(event.Members)))
	})

	discordState.AddHandler(func(event *gateway.GuildMemberRemoveEvent) {
		app.messageInput.cache.Invalidate(event.GuildID.String()+" "+event.User.Username, discordState.MemberState.SearchLimit)
	})

	discordState.StateLog = func(err error) {
		slog.Error("state log", "err", err)
	}

	discordState.OnRequest = append(discordState.OnRequest, httputil.WithHeaders(http.Headers()), onRequest)
	return discordState.Open(context.TODO())
}

func onRequest(r httpdriver.Request) error {
	if req, ok := r.(*httpdriver.DefaultRequest); ok {
		slog.Debug("new HTTP request", "method", req.Method, "url", req.URL)
	}

	return nil
}

func onRaw(event *ws.RawEvent) {
	slog.Debug(
		"new raw event",
		"code", event.OriginalCode,
		"type", event.OriginalType,
		// "data", event.Raw,
	)
}

func onReadUpdate(event *read.UpdateEvent) {
	var guildNode *tview.TreeNode
	app.guildsTree.
		GetRoot().
		Walk(func(node, parent *tview.TreeNode) bool {
			switch node.GetReference() {
			case event.GuildID:
				node.SetTextStyle(app.guildsTree.getGuildNodeStyle(event.GuildID))
				guildNode = node
				return false
			case event.ChannelID:
				// private channel
				if !event.GuildID.IsValid() {
					style := app.guildsTree.getChannelNodeStyle(event.ChannelID)
					node.SetTextStyle(style)
					return false
				}
			}

			return true
		})

	if guildNode != nil {
		guildNode.Walk(func(node, parent *tview.TreeNode) bool {
			if node.GetReference() == event.ChannelID {
				node.SetTextStyle(app.guildsTree.getChannelNodeStyle(event.ChannelID))
				return false
			}

			return true
		})
	}

	app.Draw()
}

func onReady(r *gateway.ReadyEvent) {
	dmNode := tview.NewTreeNode("Direct Messages")
	root := app.guildsTree.
		GetRoot().
		ClearChildren().
		AddChild(dmNode)

	for _, folder := range r.UserSettings.GuildFolders {
		if folder.ID == 0 && len(folder.GuildIDs) == 1 {
			guild, err := discordState.Cabinet.Guild(folder.GuildIDs[0])
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

			app.guildsTree.createGuildNode(root, *guild)
		} else {
			app.guildsTree.createFolderNode(folder)
		}
	}

	app.guildsTree.SetCurrentNode(root)
	app.SetFocus(app.guildsTree)
	app.Draw()
}

func onMessageCreate(message *gateway.MessageCreateEvent) {
	if app.guildsTree.selectedChannelID == message.ChannelID {
		app.messagesList.appendMessage(message.Message)
		app.Draw()
	}

	if err := notifications.Notify(discordState, message, app.cfg); err != nil {
		slog.Error("Notification failed", "err", err)
	}
}

func onMessageUpdate(message *gateway.MessageUpdateEvent) {
	if app.guildsTree.selectedChannelID == message.ChannelID {
		onMessageDelete(&gateway.MessageDeleteEvent{ID: message.ID, ChannelID: message.ChannelID, GuildID: message.GuildID})
	}
}

func onMessageDelete(message *gateway.MessageDeleteEvent) {
	if app.guildsTree.selectedChannelID == message.ChannelID {
		messages, err := discordState.Cabinet.Messages(message.ChannelID)
		if err != nil {
			slog.Error("failed to get messages from state", "err", err, "channel_id", message.ChannelID)
			return
		}

		app.messagesList.reset()
		app.messagesList.appendMessages(messages)
		app.Draw()
	}
}
