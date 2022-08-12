package core

import (
	"context"
	"sort"
	"strings"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/rivo/tview"
)

type Core struct {
	application *tview.Application

	guildsList   *ui.GuildsList
	channelsTree *ui.ChannelsTree

	config *config.Config

	state *state.State
}

func New(token string, cfg *config.Config) *Core {
	return &Core{
		application:  tview.NewApplication(),
		guildsList:   ui.NewGuildsList(),
		channelsTree: ui.NewChannelsTree(),

		config: cfg,

		state: state.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
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
}

func (c *Core) Run() error {
	// For user accounts, all of the guilds, the client user is in, are dispatched in the READY gateway event.
	// Whereas, for bot accounts, the guilds are dispatched discretely in the GUILD_CREATE gateway events.
	if !strings.HasPrefix(c.state.Token, "Bot") {
		api.UserAgent = c.config.Identify.UserAgent
		c.state.AddHandler(c.onStateReady)
	}

	err := c.state.Open(context.Background())
	if err != nil {
		return err
	}

	c.draw()

	c.application.EnableMouse(true)

	return c.application.Run()
}

func (c *Core) draw() {
	left := tview.NewFlex()
	left.SetDirection(tview.FlexRow)
	left.AddItem(c.guildsList, 10, 1, false)
	left.AddItem(c.channelsTree, 0, 1, false)

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)

	main := tview.NewFlex()
	main.AddItem(left, 0, 1, false)
	main.AddItem(right, 0, 4, false)

	c.application.SetRoot(main, true)
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
		c.guildsList.AddItem(g.Name, "", 0, nil)
	}
}
