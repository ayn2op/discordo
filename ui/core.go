package ui

import (
	"context"
	"strings"

	"github.com/ayntgl/discordo/config"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	lua "github.com/yuin/gopher-lua"
)

type Core struct {
	Application   *tview.Application
	MainFlex      *tview.Flex
	GuildsTree    *GuildsTree
	ChannelsTree  *ChannelsTree
	MessagesPanel *MessagesPanel
	MessageInput  *MessageInput

	Config *config.Config
	State  *state.State
}

func NewCore(token string, cfg *config.Config) *Core {
	c := &Core{
		Config: cfg,
	}

	c.Application = tview.NewApplication()
	c.Application.EnableMouse(c.Config.Bool(c.Config.State.GetGlobal("mouse")))

	c.MainFlex = tview.NewFlex()
	c.MainFlex.SetInputCapture(c.onInputCapture)

	c.GuildsTree = NewGuildsTree(c)
	c.ChannelsTree = NewChannelsTree(c)
	c.MessagesPanel = NewMessagesPanel(c)
	c.MessageInput = NewMessageInput(c)

	identifyProperties := c.Config.State.GetGlobal("identifyProperties").(*lua.LTable)
	userAgent := c.Config.String(identifyProperties.RawGetString("userAgent"))
	browser := c.Config.String(identifyProperties.RawGetString("browser"))
	browserVersion := c.Config.String(identifyProperties.RawGetString("browserVersion"))
	os := c.Config.String(identifyProperties.RawGetString("os"))

	c.State = state.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
		Token:   token,
		Intents: nil,
		Properties: gateway.IdentifyProperties{
			Browser:          browser,
			BrowserUserAgent: userAgent,
			BrowserVersion:   browserVersion,
			OS:               os,
		},
		// The official client sets the compress field as false.
		Compress: false,
	}))

	// For user accounts, all of the guilds, the user is in, are dispatched in the READY gateway event.
	// Whereas, for bot accounts, the guilds are dispatched discretely in the GUILD_CREATE gateway events.
	if !strings.HasPrefix(c.State.Token, "Bot") {
		api.UserAgent = userAgent
		c.State.AddHandler(c.onStateReady)
	}

	return c
}

func (c *Core) Start() error {
	c.State.AddHandler(c.onStateGuildCreate)
	c.State.AddHandler(c.onStateGuildDelete)
	c.State.AddHandler(c.onStateMessageCreate)
	return c.State.Open(context.Background())
}

func (c *Core) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if c.MessageInput.HasFocus() {
		return e
	}

	keysTable := c.Config.State.GetGlobal("keys").(*lua.LTable)
	applicationTable := keysTable.RawGetString("application").(*lua.LTable)

	if c.MainFlex.GetItemCount() != 0 {
		switch e.Name() {
		case c.Config.String(applicationTable.RawGetString("focusGuildsTree")):
			c.Application.SetFocus(c.GuildsTree)
			return nil
		case c.Config.String(applicationTable.RawGetString("focusChannelsTree")):
			c.Application.SetFocus(c.ChannelsTree)
			return nil
		case c.Config.String(applicationTable.RawGetString("focusMessagesPanel")):
			c.Application.SetFocus(c.MessagesPanel)
			return nil
		case c.Config.String(applicationTable.RawGetString("focusMessageInput")):
			c.Application.SetFocus(c.MessageInput)
			return nil
		}
	}

	return e
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

func (c *Core) onStateReady(r *gateway.ReadyEvent) {
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
	c.Application.SetFocus(c.GuildsTree)
}

func (c *Core) onStateGuildCreate(g *gateway.GuildCreateEvent) {
	guildNode := tview.NewTreeNode(g.Name)
	guildNode.SetReference(g.ID)

	rootNode := c.GuildsTree.GetRoot()
	rootNode.AddChild(guildNode)

	c.Application.Draw()
}

func (c *Core) onStateGuildDelete(g *gateway.GuildDeleteEvent) {
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

	c.Application.Draw()
}

func (c *Core) onStateMessageCreate(m *gateway.MessageCreateEvent) {
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
