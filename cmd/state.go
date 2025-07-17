package cmd

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/notifications"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
	"github.com/diamondburned/ningen/v3/states/read"
)

func openState(token string) error {
	props := consts.GetIdentifyProps()
	api.UserAgent = props.BrowserUserAgent
	gateway.DefaultIdentity = props
	gateway.DefaultPresence = &gateway.UpdatePresenceCommand{
		Status: app.cfg.Identify.Status,
	}

	discordState = ningen.New(token)

	// Handlers
	discordState.AddHandler(onRaw)
	discordState.AddHandler(onReady)
	discordState.AddHandler(onMessageCreate)
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

	discordState.OnRequest = append(discordState.OnRequest, httputil.WithHeaders(getHeaders(props)), onRequest)
	return discordState.Open(context.TODO())
}

func getHeaders(props gateway.IdentifyProperties) http.Header {
	header := make(http.Header)

	if rawProps, err := json.Marshal(props); err == nil {
		propsHeader := base64.StdEncoding.EncodeToString(rawProps)
		header.Set("X-Super-Properties", propsHeader)
	}

	// https://developer.mozilla.org/en-US/docs/Web/HTTP/Reference/Headers
	header.Set("Accept", "*/*")
	header.Set("Accept-Language", "en-US,en;q=0.7")
	header.Set("Origin", "https://discord.com")
	header.Set("Referer", "https://discord.com/channels/@me")
	header.Set("Sec-Fetch-Dest", "empty")
	header.Set("Sec-Fetch-Mode", "cors")
	header.Set("Sec-Fetch-Site", "same-origin")
	header.Set("X-Debug-Options", "bugReporterEnabled")
	header.Set("X-Discord-Locale", string(props.SystemLocale))
	return header
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
		"data", event.Raw,
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

func onMessageCreate(msg *gateway.MessageCreateEvent) {
	if app.guildsTree.selectedChannelID == msg.ChannelID {
		app.messagesList.createMsg(msg.Message)
		app.Draw()
	}

	if err := notifications.HandleIncomingMessage(discordState, msg, app.cfg); err != nil {
		slog.Error("Notification failed", "err", err)
	}
}

func onMessageDelete(msg *gateway.MessageDeleteEvent) {
	if app.guildsTree.selectedChannelID == msg.ChannelID {
		app.messagesList.reset()
		app.messagesList.drawMsgs(msg.ChannelID)
		app.Draw()
	}
}
