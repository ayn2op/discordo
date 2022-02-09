package ui

import (
	"sort"
	"strings"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/util"
	"github.com/rivo/tview"
)

type App struct {
	*tview.Application
	MainFlex          *tview.Flex
	GuildsList        *tview.List
	ChannelsTreeView  *tview.TreeView
	MessagesTextView  *tview.TextView
	MessageInputField *tview.InputField
	Session           *discordgo.Session
	SelectedChannel   *discordgo.Channel
	Config            *util.Config
	SelectedMessage   int
}

func NewApp() *App {
	s, _ := discordgo.New()
	return &App{
		Application:       tview.NewApplication(),
		MainFlex:          tview.NewFlex(),
		GuildsList:        tview.NewList(),
		ChannelsTreeView:  tview.NewTreeView(),
		MessagesTextView:  tview.NewTextView(),
		MessageInputField: tview.NewInputField(),

		Session:         s,
		Config:          util.LoadConfig(),
		SelectedMessage: -1,
	}
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
				OS:      "Linux",
				Browser: "Firefox",
				Device:  "",
			},
		}
		app.Session.AddHandlerOnce(app.onSessionReady)
	}

	app.Session.Token = token
	app.Session.Identify.Token = token
	app.Session.AddHandler(app.onGuildCreate)
	app.Session.AddHandler(app.onSessionMessageCreate)

	return app.Session.Open()
}

func (app *App) onSessionReady(_ *discordgo.Session, r *discordgo.Ready) {
	app.GuildsList.AddItem("Direct Messages", "", 0, nil)

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
}

func (app *App) onGuildCreate(_ *discordgo.Session, g *discordgo.GuildCreate) {
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
