package ui

import (
	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/config"
	"github.com/rivo/tview"
)

type App struct {
	*tview.Application

	ChannelsTreeView  *tview.TreeView
	MessagesTextView  *tview.TextView
	MessageInputField *tview.InputField

	Session *discordgo.Session
}

func NewApp() *App {
	s, _ := discordgo.New()
	return &App{
		Application: tview.NewApplication(),

		ChannelsTreeView:  tview.NewTreeView(),
		MessagesTextView:  tview.NewTextView(),
		MessageInputField: tview.NewInputField(),

		Session: s,
	}
}

func (app *App) Connect(token string) error {
	app.Session.Token = token
	app.Session.UserAgent = config.General.UserAgent

	app.Session.Identify.Token = token
	app.Session.Identify.Compress = false
	app.Session.Identify.Intents = 0
	app.Session.Identify.LargeThreshold = 0
	app.Session.Identify.Properties.Device = ""
	app.Session.Identify.Properties.Browser = "Firefox"
	app.Session.Identify.Properties.OS = "Linux"

	// app.Session.AddHandlerOnce(onSessionReady)
	// app.Session.AddHandler(onSessionMessageCreate)

	return app.Session.Open()
}
