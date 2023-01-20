package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/ayn2op/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
	"github.com/rivo/tview"
)

func init() {
	api.UserAgent = fmt.Sprintf("%s/0.1 (https://github.com/diamondburned/arikawa, v3)", config.Name)
	gateway.DefaultIdentity = gateway.IdentifyProperties{
		OS:      runtime.GOOS,
		Browser: config.Name,
		Device:  "",
	}
}

type State struct {
	*state.State
}

func newState(token string) *State {
	s := &State{
		State: state.New(token),
	}

	s.AddHandler(s.onReady)
	s.AddHandler(s.onMessageCreate)

	s.StateLog = s.onLog
	s.OnRequest = append(s.Client.OnRequest, s.onRequest)

	return s
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
	guildsTree.root.AddChild(dmNode)

	for _, gf := range r.UserSettings.GuildFolders {
		/// If the ID of the guild folder is zero, the guild folder only contains single guild.
		if gf.ID == 0 {
			err := guildsTree.createGuildNodeFromID(guildsTree.root, gf.GuildIDs[0])
			if err != nil {
				log.Println(err)
				continue
			}
		} else {
			guildsTree.createGuildFolderNode(guildsTree.root, gf)
		}
	}

	guildsTree.SetCurrentNode(guildsTree.root)
	app.SetFocus(guildsTree)
}

func (s *State) onMessageCreate(m *gateway.MessageCreateEvent) {
	if guildsTree.selectedChannel != nil && guildsTree.selectedChannel.ID == m.ChannelID {
		messagesText.createMessage(&m.Message)
	}
}
