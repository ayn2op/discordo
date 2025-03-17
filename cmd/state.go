package cmd

import (
	"context"
	"log/slog"
	"runtime"
	"slices"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type State struct {
	*ningen.State
}

func newState(token string) (*State, error) {
	api.UserAgent = app.cfg.UserAgent
	gateway.DefaultIdentity = gateway.IdentifyProperties{
		OS:     runtime.GOOS,
		Device: "",

		Browser:          app.cfg.Browser,
		BrowserVersion:   app.cfg.BrowserVersion,
		BrowserUserAgent: app.cfg.UserAgent,
	}

	gateway.DefaultPresence = &gateway.UpdatePresenceCommand{
		Status: app.cfg.Status,
	}

	s := &State{State: ningen.New(token)}

	// Handlers
	s.AddHandler(s.onReady)
	s.AddHandler(s.onMessageCreate)
	s.AddHandler(s.onMessageDelete)
	s.OnRequest = append(s.OnRequest, s.onRequest)

	if err := s.Open(context.TODO()); err != nil {
		return nil, err
	}

	return s, nil
}

func (s *State) onRequest(r httpdriver.Request) error {
	req, ok := r.(*httpdriver.DefaultRequest)
	if ok {
		slog.Info("new HTTP request", "method", req.Method, "path", req.URL.Path)
	}

	return nil
}

func (s *State) onReady(r *gateway.ReadyEvent) {
	root := app.guildsTree.GetRoot()
	root.ClearChildren()

	dmNode := tview.NewTreeNode("Direct Messages")
	dmNode.SetColor(tcell.GetColor(app.cfg.Theme.GuildsTree.PrivateChannelColor))
	root.AddChild(dmNode)

	// Track guilds that have a parent (folder) to add orphan channels later
	var folderGuildIds []discord.GuildID
	for _, folder := range r.UserSettings.GuildFolders {
		// Hide unnamed, single-server folders
		if folder.Name == "" && len(folder.GuildIDs) < 2 {
			continue
		}
		folderGuildIds = append(folderGuildIds, folder.GuildIDs...)

		app.guildsTree.createFolderNode(folder)
	}

	// add orphan (without folder) guilds to guilds tree
	for _, guild := range r.Guilds {
		if !slices.Contains(folderGuildIds, guild.ID) {
			app.guildsTree.createGuildNode(root, guild.Guild)
		}
	}

	app.guildsTree.SetCurrentNode(root)
	app.SetFocus(app.guildsTree)
}

func (s *State) onMessageCreate(m *gateway.MessageCreateEvent) {
	if app.guildsTree.selectedChannelID.IsValid() && app.guildsTree.selectedChannelID == m.ChannelID {
		app.messagesText.createMessage(m.Message)
	}
}

func (s *State) onMessageDelete(m *gateway.MessageDeleteEvent) {
	if app.guildsTree.selectedChannelID == m.ChannelID {
		app.messagesText.selectedMessageID = 0
		app.messagesText.Highlight()
		app.messagesText.Clear()

		app.messagesText.drawMsgs(m.ChannelID)
	}
}
