package ui

import (
	"sort"
	"strings"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	*tview.Application
	MainFlex          *tview.Flex
	GuildsList        *GuildsList
	ChannelsTreeView  *ChannelsTreeView
	MessagesTextView  *MessagesTextView
	MessageInputField *MessageInputField
	Session           *discordgo.Session
	SelectedChannel   *discordgo.Channel
	Config            *config.Config
	SelectedMessage   int
}

func NewApp(c *config.Config) *App {
	app := &App{
		MainFlex:        tview.NewFlex(),
		Config:          c,
		SelectedMessage: -1,
	}

	c.Load()

	app.GuildsList = NewGuildsList(app)
	app.ChannelsTreeView = NewChannelsTreeView(app)
	app.MessagesTextView = NewMessagesTextView(app)
	app.MessageInputField = NewMessageInputField(app)

	app.Session, _ = discordgo.New()

	app.Application = tview.NewApplication()
	app.EnableMouse(app.Config.General.Mouse)
	app.SetInputCapture(app.onInputCapture)

	return app
}

func (app *App) Connect(token string) error {
	// For user accounts, all of the guilds, the user is in, are dispatched in the READY gateway event.
	// Whereas, for bot accounts, the guilds are dispatched discretely in the GUILD_CREATE gateway events.
	if !strings.HasPrefix(token, "Bot") {
		app.Session.UserAgent = app.Config.General.UserAgent
		app.Session.Identify = discordgo.Identify{
			Compress:       false,
			LargeThreshold: 0,
			Intents:        0,
			Properties: discordgo.IdentifyProperties{
				OS:      app.Config.General.Identify.Os,
				Browser: app.Config.General.Identify.Browser,
			},
		}
		app.Session.AddHandlerOnce(app.onSessionReady)
	}

	app.Session.Token = token
	app.Session.Identify.Token = token
	app.Session.AddHandler(app.onSessionGuildCreate)
	app.Session.AddHandler(app.onSessionMessageCreate)

	return app.Session.Open()
}

func (app *App) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if app.MessageInputField.HasFocus() {
		return e
	}

	if app.MainFlex.GetItemCount() != 0 {
		switch e.Name() {
		case app.Config.Keys.ToggleGuildsList:
			app.SetFocus(app.GuildsList)
			return nil
		case app.Config.Keys.ToggleChannelsTreeView:
			app.SetFocus(app.ChannelsTreeView)
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
		AddItem(app.GuildsList, 10, 1, false).
		AddItem(app.ChannelsTreeView, 0, 1, false)
	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(app.MessagesTextView, 0, 1, false).
		AddItem(app.MessageInputField, 3, 1, false)
	app.MainFlex.
		AddItem(leftFlex, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)

	app.SetRoot(app.MainFlex, true)
}

func (app *App) onSessionReady(_ *discordgo.Session, r *discordgo.Ready) {
	sort.Slice(r.Guilds, func(a, b int) bool {
		found := false
		for _, guildID := range r.Settings.GuildPositions {
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
		app.GuildsList.AddItem(g.Name, "", 0, nil)
	}

	app.GuildsList.AddItem("Direct Messages", "", 0, nil)
}

func (app *App) onSessionGuildCreate(_ *discordgo.Session, g *discordgo.GuildCreate) {
	app.GuildsList.AddItem(g.Name, "", 0, nil)
}

func (app *App) onSessionMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if app.SelectedChannel != nil && app.SelectedChannel.ID == m.ChannelID {
		app.SelectedChannel.Messages = append(app.SelectedChannel.Messages, m.Message)
		_, err := app.MessagesTextView.Write(buildMessage(app, m.Message))
		if err != nil {
			return
		}

		if len(app.MessagesTextView.GetHighlights()) == 0 {
			app.MessagesTextView.ScrollToEnd()
		}
	}
}
