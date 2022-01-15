package ui

import (
	"fmt"
	"sort"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/config"
	"github.com/gen2brain/beeep"
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
	Config            config.Config
	SelectedMessage   int
}

func NewApp() *App {
	s, _ := discordgo.New()
	return &App{
		Application:       tview.NewApplication(),
		GuildsList:        tview.NewList(),
		ChannelsTreeView:  tview.NewTreeView(),
		MessagesTextView:  tview.NewTextView(),
		MessageInputField: tview.NewInputField(),

		Session:         s,
		SelectedMessage: -1,
	}
}

func (app *App) Connect(token string) error {
	app.Session.Token = token
	app.Session.UserAgent = app.Config.General.UserAgent

	app.Session.Identify = discordgo.Identify{
		Token:          token,
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
	app.Session.AddHandler(app.onSessionMessageCreate)

	return app.Session.Open()
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
}

func (app *App) onSessionMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if app.SelectedChannel == nil || app.SelectedChannel.ID != m.ChannelID {
		if app.Config.General.Notifications {
			for _, u := range m.Mentions {
				if u.ID == app.Session.State.User.ID {
					g, err := app.Session.State.Guild(m.GuildID)
					if err != nil {
						return
					}

					c, err := app.Session.State.Channel(m.ChannelID)
					if err != nil {
						return
					}

					go beeep.Alert(fmt.Sprintf("%s (#%s)", g.Name, c.Name), m.ContentWithMentionsReplaced(), "")
				}
			}
		}
	} else {
		app.SelectedChannel.Messages = append(app.SelectedChannel.Messages, m.Message)
		app.MessagesTextView.Write(buildMessage(app, m.Message))

		if len(app.MessagesTextView.GetHighlights()) == 0 {
			app.MessagesTextView.ScrollToEnd()
		}
	}
}
