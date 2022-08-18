package ui

import (
	"context"
	"sort"
	"strings"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	*tview.Application
	MainFlex          *tview.Flex
	GuildsTree        *GuildsTree
	ChannelsTree      *ChannelsTree
	MessagesTextView  *MessagesTextView
	MessageInputField *MessageInputField
	Config            *config.Config
	SelectedMessage   int

	State           *state.State
	SelectedChannel *discord.Channel
}

func NewApp(token string, c *config.Config) *App {
	app := &App{
		MainFlex:        tview.NewFlex(),
		Config:          c,
		SelectedMessage: -1,

		State: state.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
			Token:   token,
			Intents: nil,
			Properties: gateway.IdentifyProperties{
				OS:      c.Identify.Os,
				Browser: c.Identify.Browser,
			},
			// The official client sets the compress field as false.
			Compress: false,
		})),
	}

	app.GuildsTree = NewGuildsTree(app)
	app.ChannelsTree = NewChannelsTree(app)
	app.MessagesTextView = NewMessagesTextView(app)
	app.MessageInputField = NewMessageInputField(app)

	app.Application = tview.NewApplication()
	app.EnableMouse(app.Config.Mouse)
	app.SetInputCapture(app.onInputCapture)

	return app
}

func (app *App) Connect() error {
	// For user accounts, all of the guilds, the user is in, are dispatched in the READY gateway event.
	// Whereas, for bot accounts, the guilds are dispatched discretely in the GUILD_CREATE gateway events.
	if !strings.HasPrefix(app.State.Token, "Bot") {
		api.UserAgent = app.Config.Identify.UserAgent
		app.State.AddHandler(app.onStateReady)
	}

	app.State.AddHandler(app.onStateGuildCreate)
	// app.Session.AddHandler(app.onStateGuildDelete)
	// app.Session.AddHandler(app.onStateMessageCreate)
	return app.State.Open(context.Background())
}

func (app *App) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if app.MessageInputField.HasFocus() {
		return e
	}

	if app.MainFlex.GetItemCount() != 0 {
		switch e.Name() {
		case app.Config.Keys.ToggleGuildsList:
			app.SetFocus(app.GuildsTree)
			return nil
		case app.Config.Keys.ToggleChannelsTreeView:
			app.SetFocus(app.ChannelsTree)
			return nil
		case app.Config.Keys.ToggleMessagesTextView:
			app.SetFocus(app.MessagesTextView)
			return nil
		case app.Config.Keys.ToggleMessageInputField:
			app.SetFocus(app.MessageInputField)
			return nil
		}
	}

	return e
}

func (app *App) DrawMainFlex() {
	leftFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(app.GuildsTree, 10, 1, false).
		AddItem(app.ChannelsTree, 0, 1, false)
	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(app.MessagesTextView, 0, 1, false).
		AddItem(app.MessageInputField, 3, 1, false)
	app.MainFlex.
		AddItem(leftFlex, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)

	app.SetRoot(app.MainFlex, true)
}

func (app *App) onStateReady(r *gateway.ReadyEvent) {
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

	rootNode := app.GuildsTree.GetRoot()
	for _, g := range r.Guilds {
		guildNode := tview.NewTreeNode(g.Name)
		guildNode.SetReference(g.ID)

		rootNode.AddChild(guildNode)
	}

	app.GuildsTree.SetCurrentNode(rootNode)
	app.SetFocus(app.GuildsTree)
}

func (app *App) onStateGuildCreate(g *gateway.GuildCreateEvent) {
	rootNode := app.GuildsTree.GetRoot()
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)

	rootNode.AddChild(guildNode)
	app.Draw()
}

// func (app *App) onStateGuildDelete(g *gateway.GuildDeleteEvent) {
// 	items := app.GuildsTree.FindItems(g.Name, "", false, false)
// 	if len(items) != 0 {
// 		app.GuildsTree.RemoveItem(items[0])
// 	}

// 	app.Draw()
// }

// func (app *App) onStateMessageCreate(m *gateway.MessageCreateEvent) {
// 	if app.SelectedChannel != nil && app.SelectedChannel.ID == m.ChannelID {
// 		app.SelectedChannel.Messages = append(app.SelectedChannel.Messages, m.Message)
// 		_, err := app.MessagesTextView.Write(buildMessage(app, m.Message))
// 		if err != nil {
// 			return
// 		}

// 		if len(app.MessagesTextView.GetHighlights()) == 0 {
// 			app.MessagesTextView.ScrollToEnd()
// 		}
// 	}
// }
