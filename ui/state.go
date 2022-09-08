package ui

import (
	"context"
	"fmt"
	"runtime"
	"strings"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/rivo/tview"
)

func init() {
	api.UserAgent = fmt.Sprintf("%s/%s %s/%s", config.Name, "0.1", "arikawa", "v3")
	gateway.DefaultIdentity = gateway.IdentifyProperties{
		OS:      runtime.GOOS,
		Browser: config.Name,
	}
}

type State struct {
	*state.State
	core *Core
}

func NewState(token string, c *Core) *State {
	return &State{
		State: state.New(token),
		core:  c,
	}
}

func (s *State) Run() error {
	// Add the essential intents to the identify data for bot accounts.
	if strings.HasPrefix(s.Token, "Bot") {
		s.AddIntents(gateway.IntentGuilds | gateway.IntentGuildMessages)
	}

	s.AddHandler(s.ready)
	s.AddHandler(s.guildCreate)
	s.AddHandler(s.guildDelete)
	s.AddHandler(s.messageCreate)
	return s.Open(context.Background())
}

func (s *State) ready(r *gateway.ReadyEvent) {
	rootNode := s.core.GuildsTree.GetRoot()
	for _, gf := range r.UserSettings.GuildFolders {
		if gf.ID == 0 {
			for _, gID := range gf.GuildIDs {
				g, err := s.State.Cabinet.Guild(gID)
				if err != nil {
					return
				}

				guildNode := tview.NewTreeNode(g.Name)
				guildNode.SetReference(g.ID)
				rootNode.AddChild(guildNode)
			}
		} else {
			var b strings.Builder

			if gf.Color != discord.NullColor {
				b.WriteByte('[')
				b.WriteString(gf.Color.String())
				b.WriteByte(']')
			} else {
				b.WriteString("[#ED4245]")
			}

			if gf.Name != "" {
				b.WriteString(gf.Name)
			} else {
				b.WriteString("Folder")
			}

			b.WriteString("[-]")

			folderNode := tview.NewTreeNode(b.String())
			rootNode.AddChild(folderNode)

			for _, gID := range gf.GuildIDs {
				g, err := s.State.Cabinet.Guild(gID)
				if err != nil {
					return
				}

				guildNode := tview.NewTreeNode(g.Name)
				guildNode.SetReference(g.ID)
				folderNode.AddChild(guildNode)
			}
		}

	}

	s.core.GuildsTree.SetCurrentNode(rootNode)
	s.core.Application.SetFocus(s.core.GuildsTree)
}

func (s *State) guildCreate(g *gateway.GuildCreateEvent) {
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)

	rootNode := s.core.GuildsTree.GetRoot()
	rootNode.AddChild(guildNode)

	s.core.GuildsTree.SetCurrentNode(rootNode)
	s.core.Application.SetFocus(s.core.GuildsTree)
	s.core.Application.Draw()
}

func (s *State) guildDelete(g *gateway.GuildDeleteEvent) {
	rootNode := s.core.GuildsTree.GetRoot()
	var parentNode *tview.TreeNode
	rootNode.Walk(func(node, _ *tview.TreeNode) bool {
		if node.GetReference() == g.ID {
			parentNode = node
			return false
		}

		return true
	})

	if parentNode != nil {
		rootNode.RemoveChild(parentNode)
	}

	s.core.Application.Draw()
}

func (s *State) messageCreate(m *gateway.MessageCreateEvent) {
	if s.core.ChannelsTree.SelectedChannel != nil && s.core.ChannelsTree.SelectedChannel.ID == m.ChannelID {
		_, err := s.core.MessagesPanel.Write(buildMessage(s.core, m.Message))
		if err != nil {
			return
		}

		if len(s.core.MessagesPanel.GetHighlights()) == 0 {
			s.core.MessagesPanel.ScrollToEnd()
		}
	}
}
