package ui

import (
	"context"
	"log"
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
	guildsView FocusedID = iota
	channelsView
	messagesView
	inputView
)

// Core is responsible for the following:
// - Initialization of the application, UI elements, configuration, and state.
// - Configuration of the application and state when Run is called.
// - Management of the application and state.
type Core struct {
	App          *tview.Application
	View         *tview.Flex
	GuildsView   *GuildsView
	ChannelsView *ChannelsView
	MessagesView *MessagesView
	InputView    *InputView

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

	c.GuildsView = newGuildsView(c)
	c.ChannelsView = newChannelsView(c)
	c.MessagesView = newMessagesView(c)
	c.InputView = newInputView(c)

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
		AddItem(c.GuildsView, 10, 1, false).
		AddItem(c.ChannelsView, 0, 1, false)
	right := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(c.MessagesView, 0, 1, false).
		AddItem(c.InputView, 3, 1, false)

	c.View.AddItem(left, 0, 1, false)
	c.View.AddItem(right, 0, 4, false)

	c.App.SetRoot(c.View, true)
	c.App.SetFocus(c.GuildsView)
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
		// If the currently focused view is the guilds view (first), then focus the input view (last)
		if c.focusedID == 0 {
			c.focusedID = inputView
		} else {
			c.focusedID--
		}

		c.setFocus()
	case tcell.KeyTab:
		// If the currently focused view is the input view (last), then focus the guilds view (first)
		if c.focusedID == inputView {
			c.focusedID = guildsView
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
	case guildsView:
		p = c.GuildsView
	case channelsView:
		p = c.ChannelsView
	case messagesView:
		p = c.MessagesView
	case inputView:
		p = c.InputView
	}

	c.App.SetFocus(p)
}

func (c *Core) onReady(r *gateway.ReadyEvent) {
	root := c.GuildsView.GetRoot()
	for _, gf := range r.UserSettings.GuildFolders {
		if gf.ID == 0 {
			for _, gID := range gf.GuildIDs {
				g, err := c.State.Cabinet.Guild(gID)
				if err != nil {
					log.Println(err)
					continue
				}

				guildNode := tview.NewTreeNode(g.Name)
				guildNode.SetReference(g.ID)
				root.AddChild(guildNode)
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
			root.AddChild(folderNode)

			for _, gID := range gf.GuildIDs {
				g, err := c.State.Cabinet.Guild(gID)
				if err != nil {
					log.Println(err)
					continue
				}

				guildNode := tview.NewTreeNode(g.Name)
				guildNode.SetReference(g.ID)
				folderNode.AddChild(guildNode)
			}
		}

	}

	c.GuildsView.SetCurrentNode(root)
	c.App.SetFocus(c.GuildsView)
}

func (c *Core) onGuildCreate(g *gateway.GuildCreateEvent) {
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)

	rootNode := c.GuildsView.GetRoot()
	rootNode.AddChild(guildNode)

	c.GuildsView.SetCurrentNode(rootNode)
	c.App.SetFocus(c.GuildsView)
	c.App.Draw()
}

func (c *Core) onGuildDelete(g *gateway.GuildDeleteEvent) {
	rootNode := c.GuildsView.GetRoot()
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
	if c.ChannelsView.selectedChannel != nil && c.ChannelsView.selectedChannel.ID == m.ChannelID {
		_, err := c.MessagesView.Write(buildMessage(c, m.Message))
		if err != nil {
			return
		}

		if len(c.MessagesView.GetHighlights()) == 0 {
			c.MessagesView.ScrollToEnd()
		}
	}
}
