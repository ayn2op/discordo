package cmd

import (
	"context"
	"log"
	"runtime"

	"github.com/ayn2op/discordo/internal/constants"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
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
	*state.State
}

func openState(token string) error {
	discordState = &State{
		State: state.New(token),
	}

	// Handlers
	discordState.AddHandler(discordState.onReady)
	discordState.AddHandler(discordState.onMessageCreate)
	discordState.AddHandler(discordState.onMessageDelete)

	discordState.StateLog = discordState.onLog
	discordState.OnRequest = append(discordState.Client.OnRequest, discordState.onRequest)

	return discordState.Open(context.TODO())
}

func (s *State) onLog(err error) {
	log.Printf("%s\n", err)
}

func (s *State) onRequest(r httpdriver.Request) error {
	req, ok := r.(*httpdriver.DefaultRequest)
	if ok {
		log.Printf("method = %s; url = %s\n", req.Method, req.URL)
	}

	return nil
}

func (s *State) onReady(r *gateway.ReadyEvent) {
	dmNode := tview.NewTreeNode("Direct Messages")
	mainFlex.guildsTree.root.AddChild(dmNode)

	for _, gf := range r.UserSettings.GuildFolders {
		/// If the ID of the guild folder is zero, the guild folder only contains single guild.
		if gf.ID == 0 {
			g, err := s.Cabinet.Guild(gf.GuildIDs[0])
			if err != nil {
				log.Println(err)
				continue
			}

			mainFlex.guildsTree.createGuildNode(mainFlex.guildsTree.root, *g)
		} else {
			mainFlex.guildsTree.createGuildFolderNode(mainFlex.guildsTree.root, gf)
		}
	}

	mainFlex.guildsTree.SetCurrentNode(mainFlex.guildsTree.root)
	app.SetFocus(mainFlex.guildsTree)
}

func (s *State) onMessageCreate(m *gateway.MessageCreateEvent) {
	if mainFlex.guildsTree.selectedChannelID.IsValid() && mainFlex.guildsTree.selectedChannelID == m.ChannelID {
		mainFlex.messagesText.createMessage(m.Message)
	}
}

func (s *State) onMessageDelete(m *gateway.MessageDeleteEvent) {
	if mainFlex.guildsTree.selectedChannelID == m.ChannelID {
		mainFlex.messagesText.selectedMessage = -1
		mainFlex.messagesText.Highlight()
		mainFlex.messagesText.Clear()

		mainFlex.messagesText.drawMsgs(m.ChannelID)
	}
}
