package ui

import (
	"context"
	"strings"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type FocusedID int

const (
	guildsTree FocusedID = iota
	channelsTree
	messagesPanel
	messageInput
)

// Core is responsible for the following:
// - Initialization of the application, UI elements, configuration, and state.
// - Configuration of the application and state when Run is called.
// - Management of the application and state.
type Core struct {
	App           *tview.Application
	View          *tview.Flex
	GuildsTree    *GuildsTree
	ChannelsTree  *ChannelsTree
	MessagesPanel *MessagesPanel
	MessageInput  *MessageInput

	Config *config.Config
	State  *state.State

	focusedID FocusedID
}

func NewCore(cfg *config.Config) *Core {
	c := &Core{
		Config: cfg,
	}

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(cfg.Theme.Background)
	tview.Styles.BorderColor = tcell.GetColor(cfg.Theme.Border)
	tview.Styles.TitleColor = tcell.GetColor(cfg.Theme.Title)

	c.App = tview.NewApplication()
	c.App.EnableMouse(c.Config.Mouse)
	c.App.SetBeforeDrawFunc(c.onAppBeforeDraw)

	c.View = tview.NewFlex()
	c.View.SetInputCapture(c.onViewInputCapture)

	c.GuildsTree = NewGuildsTree(c)
	c.ChannelsTree = NewChannelsTree(c)
	c.MessagesPanel = NewMessagesPanel(c)
	c.MessageInput = NewMessageInput(c)

	return c
}

func (c *Core) Run(token string) error {
	c.State = state.New(token)
	c.State.AddHandler(c.onReady)
	c.State.AddHandler(c.onGuildCreate)
	c.State.AddHandler(c.onGuildDelete)
	c.State.AddHandler(c.onMessageCreate)
	return c.State.Open(context.Background())
}

func (c *Core) Draw() {
	left := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(c.GuildsTree, 10, 1, false).
		AddItem(c.ChannelsTree, 0, 1, false)
	right := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(c.MessagesPanel, 0, 1, false).
		AddItem(c.MessageInput, 3, 1, false)

	c.View.AddItem(left, 0, 1, false)
	c.View.AddItem(right, 0, 4, false)

	c.App.SetRoot(c.View, true)
	c.App.SetFocus(c.GuildsTree)
}

func (c *Core) onAppBeforeDraw(screen tcell.Screen) bool {
	if c.Config.Theme.Background == "default" {
		screen.Clear()
	}

	return false
}

func (c *Core) onViewInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
	case tcell.KeyEsc:
		c.focusedID = 0
	case tcell.KeyBacktab:
		// If the currently focused widget is the guilds tree widget (first), then focus the message input widget (last)
		if c.focusedID == 0 {
			c.focusedID = messageInput
		} else {
			c.focusedID--
		}

		c.setFocus()
	case tcell.KeyTab:
		// If the currently focused widget is the message input widget (last), then focus the guilds tree widget (first)
		if c.focusedID == messageInput {
			c.focusedID = guildsTree
		} else {
			c.focusedID++
		}

		c.setFocus()
	}

	return event
}

func (c *Core) setFocus() {
	var p tview.Primitive
	switch c.focusedID {
	case guildsTree:
		p = c.GuildsTree
	case channelsTree:
		p = c.ChannelsTree
	case messagesPanel:
		p = c.MessagesPanel
	case messageInput:
		p = c.MessageInput
	}

	c.App.SetFocus(p)
}

func (c *Core) onReady(r *gateway.ReadyEvent) {
	rootNode := c.GuildsTree.GetRoot()
	for _, gf := range r.UserSettings.GuildFolders {
		if gf.ID == 0 {
			for _, gID := range gf.GuildIDs {
				g, err := c.State.Cabinet.Guild(gID)
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
				g, err := c.State.Cabinet.Guild(gID)
				if err != nil {
					return
				}

				guildNode := tview.NewTreeNode(g.Name)
				guildNode.SetReference(g.ID)
				folderNode.AddChild(guildNode)
			}
		}

	}

	c.GuildsTree.SetCurrentNode(rootNode)
	c.App.SetFocus(c.GuildsTree)
}

func (c *Core) onGuildCreate(g *gateway.GuildCreateEvent) {
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)

	rootNode := c.GuildsTree.GetRoot()
	rootNode.AddChild(guildNode)

	c.GuildsTree.SetCurrentNode(rootNode)
	c.App.SetFocus(c.GuildsTree)
	c.App.Draw()
}

func (c *Core) onGuildDelete(g *gateway.GuildDeleteEvent) {
	rootNode := c.GuildsTree.GetRoot()
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

	c.App.Draw()
}

func (c *Core) onMessageCreate(m *gateway.MessageCreateEvent) {
	if c.ChannelsTree.SelectedChannel != nil && c.ChannelsTree.SelectedChannel.ID == m.ChannelID {
		_, err := c.MessagesPanel.Write(buildMessage(c, m.Message))
		if err != nil {
			return
		}

		if len(c.MessagesPanel.GetHighlights()) == 0 {
			c.MessagesPanel.ScrollToEnd()
		}
	}
}
