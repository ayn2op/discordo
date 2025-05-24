package cmd

import (
	"context"
	"log/slog"
	"runtime"
	"slices"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const userAgent = config.Name + "/0.1 (https://github.com/diamondburned/arikawa, v3)"

func init() {
	api.UserAgent = userAgent
	gateway.DefaultIdentity = gateway.IdentifyProperties{
		OS:     runtime.GOOS,
		Device: "",

		Browser:          config.Name,
		BrowserUserAgent: userAgent,
	}
}

type State struct {
	*ningen.State
	cfg *config.Config
	app *tview.Application
}

func openState(token string, app *tview.Application, cfg *config.Config) error {
	discordState = &State{
		State: ningen.New(token),
		cfg:   cfg,
		app:   app,
	}

	// Handlers
	discordState.AddHandler(discordState.onReady)
	discordState.AddHandler(discordState.onMessageCreate)
	discordState.AddHandler(discordState.onMessageDelete)

	discordState.OnRequest = append(discordState.Client.OnRequest, discordState.onRequest)
	return discordState.Open(context.TODO())
}

func (s *State) onRequest(r httpdriver.Request) error {
	req, ok := r.(*httpdriver.DefaultRequest)
	if ok {
		slog.Info("new HTTP request", "method", req.Method, "path", req.URL.Path)
	}

	return nil
}

func (s *State) onReady(r *gateway.ReadyEvent) {
	root := layout.guildsTree.GetRoot()
	dmNode := tview.NewTreeNode("Direct Messages")
	dmNode.SetColor(tcell.GetColor(s.cfg.Theme.GuildsTree.PrivateChannelColor))
	root.AddChild(dmNode)

	// Track guilds that have a parent (folder) to add orphan channels later
	var folderGuildIds []discord.GuildID
	for _, folder := range r.UserSettings.GuildFolders {
		// Hide unnamed, single-server folders
		if folder.Name == "" && len(folder.GuildIDs) < 2 {
			continue
		}
		folderGuildIds = append(folderGuildIds, folder.GuildIDs...)

		layout.guildsTree.createFolderNode(folder)
	}

	// add orphan (without folder) guilds to guilds tree
	for _, guild := range r.Guilds {
		if !slices.Contains(folderGuildIds, guild.ID) {
			layout.guildsTree.createGuildNode(root, guild.Guild)
		}
	}

	layout.guildsTree.SetCurrentNode(root)
	s.app.SetFocus(layout.guildsTree)
}

func (s *State) onMessageCreate(m *gateway.MessageCreateEvent) {
	if layout.guildsTree.selectedChannelID.IsValid() && layout.guildsTree.selectedChannelID == m.ChannelID {
		layout.messagesText.createMessage(m.Message)
	}
}

func (s *State) onMessageDelete(m *gateway.MessageDeleteEvent) {
	if layout.guildsTree.selectedChannelID == m.ChannelID {
		layout.messagesText.selectedMessageID = 0
		layout.messagesText.Highlight()
		layout.messagesText.Clear()

		layout.messagesText.drawMsgs(m.ChannelID)
	}
}
