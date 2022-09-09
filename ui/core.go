package ui

import (
	"github.com/ayntgl/discordo/config"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type focused int

const (
	guildsTree focused = iota
	channelsTree
	messagesPanel
	messageInput
)

// Core is responsible for the following:
// - Initialization of the application, UI elements, configuration, and state.
// - Configuration of the application and state when Run is called.
// - Management of the application and state.
type Core struct {
	Application   *tview.Application
	MainFlex      *tview.Flex
	GuildsTree    *GuildsTree
	ChannelsTree  *ChannelsTree
	MessagesPanel *MessagesPanel
	MessageInput  *MessageInput

	Config *config.Config
	State  *State

	focused focused
}

func NewCore(cfg *config.Config) *Core {
	c := &Core{
		Application: tview.NewApplication(),
		MainFlex:    tview.NewFlex(),

		Config: cfg,
	}

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfg.Theme.Background)
	tview.Styles.BorderColor = tcell.GetColor(cfg.Theme.Border)
	tview.Styles.TitleColor = tcell.GetColor(cfg.Theme.Title)

	c.Application.EnableMouse(c.Config.Mouse)
	c.Application.SetInputCapture(c.onInputCapture)
	c.Application.SetBeforeDrawFunc(c.beforeDraw)

	c.GuildsTree = NewGuildsTree(c)
	c.ChannelsTree = NewChannelsTree(c)
	c.MessagesPanel = NewMessagesPanel(c)
	c.MessageInput = NewMessageInput(c)
	return c
}

func (c *Core) Run(token string) error {
	c.State = NewState(token, c)
	return c.State.Run()
}

func (c *Core) DrawMainFlex() {
	leftFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(c.GuildsTree, 10, 1, false).
		AddItem(c.ChannelsTree, 0, 1, false)
	rightFlex := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(c.MessagesPanel, 0, 1, false).
		AddItem(c.MessageInput, 3, 1, false)
	c.MainFlex.
		AddItem(leftFlex, 0, 1, false).
		AddItem(rightFlex, 0, 4, false)
}

func (c *Core) beforeDraw(screen tcell.Screen) bool {
	if c.Config.Theme.Background == "default" {
		screen.Clear()
	}

	return false
}

func (c *Core) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	// If the main flex is nil, that is, it is not initialized yet, then the login form is currently focused.
	if c.MainFlex == nil {
		return event
	}

	switch event.Key() {
	case tcell.KeyEsc:
		c.focused = 0
	case tcell.KeyBacktab:
		// If the currently focused widget is the guilds tree widget (first), then focus the message input widget (last)
		if c.focused == 0 {
			c.focused = messageInput
		} else {
			c.focused--
		}

		c.setFocus()
	case tcell.KeyTab:
		// If the currently focused widget is the message input widget (last), then focus the guilds tree widget (first)
		if c.focused == messageInput {
			c.focused = guildsTree
		} else {
			c.focused++
		}

		c.setFocus()
	}

	return event
}

func (c *Core) setFocus() {
	var p tview.Primitive
	switch c.focused {
	case guildsTree:
		p = c.GuildsTree
	case channelsTree:
		p = c.ChannelsTree
	case messagesPanel:
		p = c.MessagesPanel
	case messageInput:
		p = c.MessageInput
	}

	c.Application.SetFocus(p)
}
