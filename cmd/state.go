package cmd

import (
	"context"
	"log/slog"
	"runtime"
	"slices"

	"github.com/ayn2op/discordo/internal/constants"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/state/store"
	"github.com/diamondburned/arikawa/v3/state/store/defaultstore"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/diamondburned/ningen/v3"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	api.UserAgent = constants.UserAgent
	gateway.DefaultIdentity = gateway.IdentifyProperties{
		OS:      runtime.GOOS,
		Browser: constants.Name,
		Device:  "",
	}
}

type State struct {
	*ningen.State
}

type OtherState struct {
	*state.State
}

func openState(token string) error {
	discordState = &State{
		State: ningen.FromState(
			state.NewWithStore(
				token, 
				&store.Cabinet{
					MeStore:         defaultstore.NewMe(),
					ChannelStore:    defaultstore.NewChannel(),
					EmojiStore:      defaultstore.NewEmoji(),
					GuildStore:      defaultstore.NewGuild(),
					MemberStore:     defaultstore.NewMember(),
					MessageStore:    defaultstore.NewMessage(int(cfg.MessagesLimit)),
					PresenceStore:   defaultstore.NewPresence(),
					RoleStore:       defaultstore.NewRole(),
					VoiceStateStore: defaultstore.NewVoiceState(),
				},
			),
		),
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
	root := mainFlex.guildsTree.GetRoot()
	dmNode := tview.NewTreeNode("Direct Messages")
	dmNode.SetColor(tcell.GetColor(cfg.Theme.GuildsTree.PrivateChannelColor))
	root.AddChild(dmNode)

	// Track guilds that have a parent (folder) to add orphan channels later
	var folderGuildIds []discord.GuildID
	for _, folder := range r.UserSettings.GuildFolders {
		// Hide unnamed, single-server folders
		if folder.Name == "" && len(folder.GuildIDs) < 2 {
			continue
		}
		folderGuildIds = append(folderGuildIds, folder.GuildIDs...)

		mainFlex.guildsTree.createFolderNode(folder)
	}

	// add orphan (without folder) guilds to guilds tree
	for _, guild := range r.Guilds {
		if !slices.Contains(folderGuildIds, guild.ID) {
			mainFlex.guildsTree.createGuildNode(root, guild.Guild)
		}
	}

	mainFlex.guildsTree.SetCurrentNode(root)
	app.SetFocus(mainFlex.guildsTree)
}

func (s *State) onMessageCreate(m *gateway.MessageCreateEvent) {
	if mainFlex.guildsTree.selectedChannelID.IsValid() && mainFlex.guildsTree.selectedChannelID == m.ChannelID {
		mainFlex.messagesText.createMessage(m.Message)
	}
}

func (s *State) onMessageDelete(m *gateway.MessageDeleteEvent) {
	if mainFlex.guildsTree.selectedChannelID == m.ChannelID {
		mainFlex.messagesText.selectedMessageID = 0
		mainFlex.messagesText.Highlight()
		mainFlex.messagesText.Clear()

		mainFlex.messagesText.drawMsgs(m.ChannelID)
	}
}
