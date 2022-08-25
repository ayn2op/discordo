package ui

import (
	"context"
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
	MessagesPanel     *MessagesPanel
	MessageInputField *MessageInput

	Config *config.Config
	State  *state.State
}

func NewApp(token string, c *config.Config) *App {
	app := &App{
		Application: tview.NewApplication(),
		MainFlex:    tview.NewFlex(),
		Config:      c,

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
	app.MessagesPanel = NewMessagesPanel(app)
	app.MessageInputField = NewMessageInput(app)
	app.EnableMouse(app.Config.Mouse)
	app.MainFlex.SetInputCapture(app.onInputCapture)

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
	app.State.AddHandler(app.onStateGuildDelete)
	app.State.AddHandler(app.onStateMessageCreate)
	return app.State.Open(context.Background())
}

func (app *App) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if app.MessageInputField.HasFocus() {
		return e
	}

	if app.MainFlex.GetItemCount() != 0 {
		switch e.Name() {
		case app.Config.Keys.ToggleGuildsTree:
			app.SetFocus(app.GuildsTree)
			return nil
		case app.Config.Keys.ToggleChannelsTree:
			app.SetFocus(app.ChannelsTree)
			return nil
		case app.Config.Keys.ToggleMessagesPanel:
			app.SetFocus(app.MessagesPanel)
			return nil
		case app.Config.Keys.ToggleMessageInput:
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
		AddItem(app.MessagesPanel, 0, 1, false).
		AddItem(app.MessageInputField, 3, 1, false)
	app.MainFlex.
		AddItem(leftFlex, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)
}

func (app *App) onStateReady(r *gateway.ReadyEvent) {
	rootNode := app.GuildsTree.GetRoot()
	for _, gf := range r.UserSettings.GuildFolders {
		if gf.ID == 0 {
			for _, gID := range gf.GuildIDs {
				g, err := app.State.Cabinet.Guild(gID)
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
				g, err := app.State.Cabinet.Guild(gID)
				if err != nil {
					return
				}

				guildNode := tview.NewTreeNode(g.Name)
				guildNode.SetReference(g.ID)
				folderNode.AddChild(guildNode)
			}
		}

	}

	app.GuildsTree.SetCurrentNode(rootNode)
	app.SetFocus(app.GuildsTree)
}

func (app *App) onStateGuildCreate(g *gateway.GuildCreateEvent) {
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)

	rootNode := app.GuildsTree.GetRoot()
	rootNode.AddChild(guildNode)

	app.Draw()
}

func (app *App) onStateGuildDelete(g *gateway.GuildDeleteEvent) {
	rootNode := app.GuildsTree.GetRoot()
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

	app.Draw()
}

func (app *App) onStateMessageCreate(m *gateway.MessageCreateEvent) {
	if app.ChannelsTree.SelectedChannel != nil && app.ChannelsTree.SelectedChannel.ID == m.ChannelID {
		_, err := app.MessagesPanel.Write(buildMessage(app, m.Message))
		if err != nil {
			return
		}

		if len(app.MessagesPanel.GetHighlights()) == 0 {
			app.MessagesPanel.ScrollToEnd()
		}
	}
}
