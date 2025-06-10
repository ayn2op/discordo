package cmd

import (
	"context"
	"log/slog"
	"runtime"

	"github.com/ayn2op/discordo/internal/notifications"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/arikawa/v3/utils/ws"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v2"
)

type state struct {
	*ningen.State
}

func openState(token string) error {
	api.UserAgent = app.cfg.Identify.UserAgent
	gateway.DefaultIdentity = gateway.IdentifyProperties{
		OS:     runtime.GOOS,
		Device: "",

		Browser:          app.cfg.Identify.Browser,
		BrowserVersion:   app.cfg.Identify.BrowserVersion,
		BrowserUserAgent: app.cfg.Identify.UserAgent,
	}

	gateway.DefaultPresence = &gateway.UpdatePresenceCommand{
		Status: app.cfg.Identify.Status,
	}

	discordState = &state{
		State: ningen.New(token),
	}

	// Handlers
	discordState.AddHandler(discordState.onReady)
	discordState.AddHandler(discordState.onMessageCreate)
	discordState.AddHandler(discordState.onMessageDelete)

	discordState.AddHandler(func(_ *gateway.GuildMembersChunkEvent) {
		app.messagesText.setFetchingChunk(false)
	})

	discordState.AddHandler(func(event *ws.RawEvent) {
		slog.Debug(
			"new raw event",
			"code",
			event.OriginalCode,
			"type",
			event.OriginalType,
			"data",
			event.Raw,
		)
	})

	discordState.StateLog = func(err error) {
		slog.Error("state log", "err", err)
	}

	discordState.OnRequest = append(discordState.OnRequest, discordState.onRequest)

	return discordState.Open(context.TODO())
}

func (s *state) onRequest(r httpdriver.Request) error {
	req, ok := r.(*httpdriver.DefaultRequest)
	if ok {
		slog.Debug("new HTTP request", "method", req.Method, "url", req.URL)
	}

	return nil
}

func (s *state) onReady(ready *gateway.ReadyEvent) {
	root := app.guildsTree.GetRoot()
	root.ClearChildren()

	dmNode := tview.NewTreeNode("Direct Messages")
	dmNode.SetColor(tcell.GetColor(app.cfg.Theme.GuildsTree.PrivateChannelColor))
	root.AddChild(dmNode)

	for _, folder := range ready.UserSettings.GuildFolders {
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

func (s *state) onMessageCreate(msg *gateway.MessageCreateEvent) {
	if app.guildsTree.selectedChannelID.IsValid() &&
		app.guildsTree.selectedChannelID == msg.ChannelID {
		app.messagesText.createMsg(msg.Message)
	}

	if err := notifications.HandleIncomingMessage(*s.State, msg, app.cfg); err != nil {
		slog.Error("Notification failed", "err", err)
	}
}

func (s *state) onMessageDelete(msg *gateway.MessageDeleteEvent) {
	if app.guildsTree.selectedChannelID == msg.ChannelID {
		app.messagesText.selectedMessageID = 0
		app.messagesText.Highlight()
		app.messagesText.Clear()

		app.messagesText.drawMsgs(msg.ChannelID)
	}
}
