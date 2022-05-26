package ui

import (
	"context"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type App struct {
	*tview.Application

	config *config.Config
	state  *state.State

	mainFlex          *tview.Flex
	guildsList        *GuildsList
	channelsTree      *ChannelsTree
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
}

func NewApp(token string, cfg *config.Config) *App {
	a := &App{
		Application: tview.NewApplication(),
		config:      cfg,
	}

	api.UserAgent = cfg.Identify.UserAgent
	a.state = state.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
		Token: token,
		Properties: gateway.IdentifyProperties{
			OS:      cfg.Identify.Os,
			Browser: cfg.Identify.Browser,
			Device:  "",
		},
		Presence: gateway.DefaultPresence,
	}))

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfg.Theme.Background)
	tview.Styles.BorderColor = tcell.GetColor(cfg.Theme.BorderForeground)
	tview.Styles.TitleColor = tcell.GetColor(cfg.Theme.TitleForeground)
	// Rounded borders
	tview.Borders.Vertical = '│'
	tview.Borders.Horizontal = '─'
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'

	a.mainFlex = tview.NewFlex()
	a.guildsList = NewGuildsList(a)
	a.channelsTree = NewChannelsTreeView(a)
	a.messagesTextView = NewMessagesTextView()
	a.messageInputField = NewMessageInputField(a)

	a.drawMainFlex()
	a.SetRoot(a.mainFlex, true)
	a.EnableMouse(cfg.Mouse)

	return a
}

func (a *App) Start() error {
	a.state.AddHandler(a.onSessionReady)
	err := a.state.Open(context.Background())
	if err != nil {
		return err
	}

	return a.Run()
}

func (a *App) drawMainFlex() {
	left := tview.NewFlex()
	left.SetDirection(tview.FlexRow)
	left.AddItem(a.guildsList, 0, 1, false)
	left.AddItem(a.channelsTree, 0, 2, false)

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(a.messagesTextView, 0, 1, false)
	right.AddItem(a.messageInputField, 3, 0, false)

	a.mainFlex.AddItem(left, 0, 1, false)
	a.mainFlex.AddItem(right, 0, 4, false)
}

func (a *App) onSessionReady(r *gateway.ReadyEvent) {
	for _, g := range r.Guilds {
		a.guildsList.AddItem(g.Name, "", 0, nil)
	}
}
