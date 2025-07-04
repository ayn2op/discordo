package cmd

import (
	"context"
	"log/slog"

	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/notifications"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
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

	discordState.AddHandler(onReady)
	discordState.AddHandler(onMessageCreate)
	discordState.AddHandler(onMessageDelete)

	discordState.AddHandler(func(event *gateway.GuildMembersChunkEvent) {
		app.messagesList.setFetchingChunk(false, uint(len(event.Members)))
	})

	discordState.AddHandler(func(event *gateway.GuildMemberRemoveEvent) {
		app.messageInput.cache.Invalidate(event.GuildID.String()+" "+event.User.Username, discordState.MemberState.SearchLimit)
	})

	discordState.AddHandler(func(event *ws.RawEvent) {
		slog.Debug(
			"new raw event",
			"code", event.OriginalCode,
			"type", event.OriginalType,
			"data", event.Raw,
		)
	})

	discordState.AddHandler(func(event *read.UpdateEvent) {
		app.QueueUpdateDraw(func() {
			app.guildsTree.refreshChannelDisplay(event.ReadState.ChannelID)
		})
	})

	discordState.StateLog = func(err error) {
		slog.Error("state log", "err", err)
	}

	discordState.OnRequest = append(discordState.OnRequest, onRequest)

	return discordState.Open(context.TODO())
}

func onRequest(r httpdriver.Request) error {
	req, ok := r.(*httpdriver.DefaultRequest)
	if ok {
		slog.Debug("new HTTP request", "method", req.Method, "url", req.URL)
	}

	return nil
}

func onReady(r *gateway.ReadyEvent) {
	root := app.guildsTree.GetRoot()
	root.ClearChildren()

	style := app.cfg.Theme.GuildsTree.PrivateChannelStyle.Style
	dmNode := tview.NewTreeNode("Direct Messages").
		SetTextStyle(style).
		SetSelectedTextStyle(style.Reverse(true))
	root.AddChild(dmNode)

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
}

func onMessageCreate(msg *gateway.MessageCreateEvent) {
	if app.guildsTree.selectedChannelID.IsValid() &&
		app.guildsTree.selectedChannelID == msg.ChannelID {
		app.messagesList.createMsg(msg.Message)
		app.Draw()
	} else {
		if msg.Author.ID != discordState.Ready().User.ID {
			mentions := 0
			if msg.MentionEveryone {
				mentions = 1
			} else {
				userID := discordState.Ready().User.ID
				for _, mention := range msg.Mentions {
					if mention.ID == userID {
						mentions = 1
						break
					}
				}
			}
			discordState.ReadState.MarkUnread(msg.ChannelID, msg.ID, mentions)
			app.Draw()
		}
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
