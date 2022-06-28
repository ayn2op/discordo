package ui

import (
	"sort"
	"strings"

	"github.com/ayntgl/astatine"
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
	Session           *astatine.Session
	SelectedChannel   *astatine.Channel
	Config            *config.Config
	SelectedMessage   int
}

func NewApp(token string, c *config.Config) *App {
	app := &App{
		MainFlex:        tview.NewFlex(),
		Session:         astatine.New(token),
		Config:          c,
		SelectedMessage: -1,
	}

	app.GuildsList = NewGuildsList(app)
	app.ChannelsTreeView = NewChannelsTreeView(app)
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
	if !strings.HasPrefix(app.Session.Identify.Token, "Bot") {
		app.Session.UserAgent = app.Config.Identify.UserAgent
		app.Session.Identify.Compress = false
		app.Session.Identify.LargeThreshold = 0
		app.Session.Identify.Intents = 0
		app.Session.Identify.Properties = astatine.IdentifyProperties{
			OS:      app.Config.Identify.Os,
			Browser: app.Config.Identify.Browser,
		}
		app.Session.AddHandlerOnce(app.onSessionReady)
	}

	app.Session.AddHandler(app.onSessionGuildCreate)
	app.Session.AddHandler(app.onSessionGuildDelete)
	app.Session.AddHandler(app.onSessionMessageCreate)
	return app.Session.Open()
}

func (app *App) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if app.MessageInputField.HasFocus() {
		return e
	}

	if app.MainFlex.GetItemCount() != 0 {
		switch e.Name() {
		// Avoid focusing what is already focused so that the widgets can use shortcuts
		case app.Config.Keys.ToggleGuildsList:
			if app.GuildsList.HasFocus() {
				return e
			}
			app.SetFocus(app.GuildsList)
			return nil
		case app.Config.Keys.ToggleChannelsTreeView:
			if app.ChannelsTreeView.HasFocus() {
				return e
			}
			app.SetFocus(app.ChannelsTreeView)
			return nil
		case app.Config.Keys.ToggleMessagesTextView:
			if app.ChannelsTreeView.HasFocus() {
				return e
			}
			app.SetFocus(app.MessagesTextView)
			return nil
		case app.Config.Keys.ToggleMessageInputField:
			if app.MessageInputField.HasFocus() {
				return e
			}
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

func (app *App) onSessionReady(_ *astatine.Session, r *astatine.Ready) {
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

func (app *App) onSessionGuildCreate(_ *astatine.Session, g *astatine.GuildCreate) {
	app.GuildsList.AddItem(g.Name, "", 0, nil)
	app.Draw()
}

func (app *App) onSessionGuildDelete(_ *astatine.Session, g *astatine.GuildDelete) {
	items := app.GuildsList.FindItems(g.BeforeDelete.Name, "", false, false)
	if len(items) != 0 {
		app.GuildsList.RemoveItem(items[0])
	}

	app.Draw()
}

func (app *App) onSessionMessageCreate(_ *astatine.Session, m *astatine.MessageCreate) {
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
