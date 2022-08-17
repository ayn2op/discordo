package ui

import (
	"context"
	"sort"
	"strings"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/rivo/tview"
)

type Core struct {
	Application   *tview.Application
	Flex          *tview.Flex
	GuildsList    *GuildsList
	ChannelsTree  *ChannelsTree
	MessagesPanel *MessagesPanel
	MessageInput  *MessageInput

	config *config.Config
	State  *state.State
}

func NewCore(token string, cfg *config.Config) *Core {
	c := &Core{
		Application: tview.NewApplication(),
		config:      cfg,

		State: state.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
			Token:   token,
			Intents: nil,
			Properties: gateway.IdentifyProperties{
				OS:      cfg.Identify.Os,
				Browser: cfg.Identify.Browser,
			},
			// The official client sets the compress field as false.
			Compress: false,
		})),
	}

	c.Application.EnableMouse(c.config.Mouse)

	c.GuildsList = NewGuildsList(c)
	c.ChannelsTree = NewChannelsTree(c)
	c.MessagesPanel = NewMessagesPanel(c)
	c.MessageInput = NewMessageInput(c)

	return c
}

func (c *Core) Connect() error {
	// For user accounts, all of the guilds, the client user is in, are dispatched in the READY gateway event.
	// Whereas, for bot accounts, the guilds are dispatched discretely in the GUILD_CREATE gateway events.
	if !strings.HasPrefix(c.State.Token, "Bot") {
		api.UserAgent = c.config.Identify.UserAgent
		c.State.AddHandler(c.onStateReady)
	}

	return c.State.Open(context.Background())
}

func (c *Core) DrawFlex() {
	left := tview.NewFlex()
	left.SetDirection(tview.FlexRow)
	left.AddItem(c.GuildsList, 10, 1, false)
	left.AddItem(c.ChannelsTree, 0, 1, false)

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(c.MessagesPanel, 0, 1, false)
	right.AddItem(c.MessageInput, 3, 1, false)

	c.Flex = tview.NewFlex()
	c.Flex.AddItem(left, 0, 1, false)
	c.Flex.AddItem(right, 0, 4, false)

	c.Application.SetRoot(c.Flex, true)
}

func (c *Core) onStateReady(r *gateway.ReadyEvent) {
	sort.Slice(r.Guilds, func(a, b int) bool {
		found := false
		for _, guildID := range r.UserSettings.GuildPositions {
			if found && guildID == r.Guilds[b].ID {
				return true
			}
			if !found && guildID == r.Guilds[a].ID {
				found = true
			}
		}

		return false
	})

	for _, g := range r.Guilds {
		c.GuildsList.AddItem(g.Name, "", 0, nil)
	}
}
