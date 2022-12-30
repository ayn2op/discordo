package main

import (
	"log"

	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/rivo/tview"
)

type State struct {
	*state.State
}

func newState(token string) *State {
	s := &State{
		State: state.New(token),
	}

	s.AddHandler(s.onReady)

	return s
}

func (s *State) onReady(r *gateway.ReadyEvent) {
	dmNode := tview.NewTreeNode("Direct Messages")
	guildsTree.root.AddChild(dmNode)

	for _, gf := range r.UserSettings.GuildFolders {
		/// If the ID of the guild folder is zero, the guild folder only contains single guild.
		if gf.ID == 0 {
			if err := guildsTree.newGuildFromID(guildsTree.root, gf.GuildIDs[0]); err != nil {
				log.Println(err)
				continue
			}
		} else {
			gfNode := tview.NewTreeNode("Folder")
			guildsTree.root.AddChild(gfNode)

			for _, gid := range gf.GuildIDs {
				if err := guildsTree.newGuildFromID(gfNode, gid); err != nil {
					log.Println(err)
					continue
				}
			}
		}
	}

	guildsTree.SetCurrentNode(guildsTree.root)
	app.SetFocus(guildsTree)
}
