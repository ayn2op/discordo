package main

import (
	"fmt"
	"log"
	"runtime"

	"github.com/ayn2op/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/rivo/tview"
)

func init() {
	api.UserAgent = fmt.Sprintf("%s/%s %s/%s", config.Name, "0.1", "arikawa", "v3")
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

	return s
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
			name := gf.Name
			if name == "" {
				name = "Folder"
			}

			gfNode := tview.NewTreeNode(name)
			guildsTree.root.AddChild(gfNode)

			for _, gid := range gf.GuildIDs {
				err := guildsTree.createGuildNodeFromID(gfNode, gid)
				if err != nil {
					log.Println(err)
					continue
				}
			}
		}
	}

	guildsTree.SetCurrentNode(guildsTree.root)
	app.SetFocus(guildsTree)
}

func (s *State) onMessageCreate(m *gateway.MessageCreateEvent) {
	if guildsTree.selectedChannel != nil && guildsTree.selectedChannel.ID == m.ChannelID {
		err := messagesText.createMessage(&m.Message)
		if err != nil {
			log.Println(err)
			return
		}
	}
}
