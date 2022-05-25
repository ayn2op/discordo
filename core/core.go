package core

import (
	"context"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type Core struct {
	config  *config.Config
	session *session.Session

	app               *tview.Application
	mainFlex          *tview.Flex
	guildsList        *tview.List
	channelsTreeView  *tview.TreeView
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
}

func New(token string, cfg *config.Config) *Core {
	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfg.Theme.Background)
	tview.Styles.BorderColor = tcell.GetColor(cfg.Theme.BorderForeground)

	// Rounded borders
	tview.Borders.Vertical = '│'
	tview.Borders.Horizontal = '─'
	tview.Borders.TopLeft = '╭'
	tview.Borders.TopRight = '╮'
	tview.Borders.BottomLeft = '╰'
	tview.Borders.BottomRight = '╯'

	c := &Core{
		config:            cfg,
		app:               tview.NewApplication(),
		mainFlex:          tview.NewFlex(),
		guildsList:        ui.NewGuildsList(),
		channelsTreeView:  ui.NewChannelsTreeView(),
		messagesTextView:  ui.NewMessagesTextView(),
		messageInputField: ui.NewMessageInputField(),
	}

	c.session = session.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
		Token: token,
		Properties: gateway.IdentifyProperties{
			OS:      cfg.Identify.Os,
			Browser: cfg.Identify.Browser,
			Device:  "",
		},
		Presence: gateway.DefaultPresence,
	}))

	c.guildsList.SetMainTextColor(tcell.GetColor(cfg.Theme.GuildsList.ItemForeground))
	c.guildsList.SetSelectedTextColor(tcell.GetColor(cfg.Theme.GuildsList.SelectedItemForeground))

	c.messageInputField.SetFieldTextColor(tcell.GetColor(cfg.Theme.MessageInputField.FieldForeground))
	c.messageInputField.SetPlaceholderStyle(tcell.StyleDefault.Foreground(tcell.GetColor(cfg.Theme.MessageInputField.PlaceholderForeground)).Background(tview.Styles.PrimitiveBackgroundColor))

	c.guildsList.SetSelectedBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	c.messageInputField.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)

	c.drawMainFlex()
	c.app.SetRoot(c.mainFlex, true)
	c.app.EnableMouse(cfg.Mouse)

	return c
}

func (c *Core) Run() error {
	c.session.AddHandler(c.onSessionReady)
	err := c.session.Open(context.Background())
	if err != nil {
		return err
	}

	return c.app.Run()
}

func (c *Core) drawMainFlex() {
	leftFlex := tview.NewFlex()
	leftFlex.SetDirection(tview.FlexRow)
	leftFlex.AddItem(c.guildsList, 0, 1, false)
	leftFlex.AddItem(c.channelsTreeView, 0, 2, false)

	rightFlex := tview.NewFlex()
	rightFlex.SetDirection(tview.FlexRow)
	rightFlex.AddItem(c.messagesTextView, 0, 1, false)
	rightFlex.AddItem(c.messageInputField, 3, 0, false)

	c.mainFlex.AddItem(leftFlex, 0, 1, false)
	c.mainFlex.AddItem(rightFlex, 0, 4, false)
}

func (c *Core) onSessionReady(r *gateway.ReadyEvent) {
	for _, g := range r.Guilds {
		c.guildsList.AddItem(g.Name, "", 0, nil)
	}
}
