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
	luar "layeh.com/gopher-luar"
)

var cfg []byte

// Core initializes the application, UI elements, configuration, session, and state. It also manages the application, session, and state.
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

func NewCore(path string) *Core {
	c := &Core{
		Application: tview.NewApplication(),
		MainFlex:    tview.NewFlex(),

		Config: config.NewConfig(path),
	}

	c.MainFlex.SetInputCapture(c.onInputCapture)
	c.GuildsTree = NewGuildsTree(c)
	c.ChannelsTree = NewChannelsTree(c)
	c.MessagesPanel = NewMessagesPanel(c)
	c.MessageInput = NewMessageInput(c)
	return c
}

func (c *Core) Run(token string) error {
	err := c.Config.Load()
	if err != nil {
		return err
	}

	c.register()
	err = c.Config.State.DoString(string(config.LuaConfig))
	if err != nil {
		return err
	}

	c.Application.EnableMouse(lua.LVAsBool(c.Config.State.GetGlobal("mouse")))

	identifyProperties, ok := c.Config.State.GetGlobal("identifyProperties").(*lua.LTable)
	if !ok {
		identifyProperties = c.Config.State.NewTable()
	}

	userAgent := lua.LVAsString(identifyProperties.RawGetString("userAgent"))

	c.State = state.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
		Token:   token,
		Intents: nil,
		Properties: gateway.IdentifyProperties{
			Browser:          lua.LVAsString(identifyProperties.RawGetString("browser")),
			BrowserVersion:   lua.LVAsString(identifyProperties.RawGetString("browserVersion")),
			BrowserUserAgent: userAgent,
			OS:               lua.LVAsString(identifyProperties.RawGetString("os")),
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

	c.State.AddHandler(c.onStateGuildCreate)
	c.State.AddHandler(c.onStateGuildDelete)
	c.State.AddHandler(c.onStateMessageCreate)
	return c.State.Open(context.Background())
}

func (c *Core) register() {
	c.Config.State.SetGlobal("key", c.Config.State.NewFunction(c.Config.KeyLua))
	// Messages panel
	c.Config.State.SetGlobal("openMessageActionsList", c.Config.State.NewFunction(c.MessagesPanel.openMessageActionsListLua))
	c.Config.State.SetGlobal("selectPreviousMessage", c.Config.State.NewFunction(c.MessagesPanel.selectPreviousMessageLua))
	c.Config.State.SetGlobal("selectNextMessage", c.Config.State.NewFunction(c.MessagesPanel.selectNextMessageLua))
	c.Config.State.SetGlobal("selectFirstMessage", c.Config.State.NewFunction(c.MessagesPanel.selectFirstMessageLua))
	c.Config.State.SetGlobal("selectLastMessage", c.Config.State.NewFunction(c.MessagesPanel.selectLastMessageLua))
	// Message input
	c.Config.State.SetGlobal("openExternalEditor", c.Config.State.NewFunction(c.MessageInput.openExternalEditorLua))
	c.Config.State.SetGlobal("pasteClipboardContent", c.Config.State.NewFunction(c.MessageInput.pasteClipboardContentLua))
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

func (c *Core) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if c.MessageInput.HasFocus() {
		return e
	}
	// If the main flex is nil, that is, it is not initialized yet, then the login form is currently focused.
	if c.MainFlex == nil {
		return e
	}

	keysTable, ok := c.Config.State.GetGlobal("keys").(*lua.LTable)
	if !ok {
		return e
	}

	applicationTable, ok := keysTable.RawGetString("application").(*lua.LTable)
	if !ok {
		return e
	}

	var fn lua.LValue
	applicationTable.ForEach(func(k, v lua.LValue) {
		keyTable := v.(*lua.LTable)
		if e.Name() == lua.LVAsString(keyTable.RawGetString("name")) {
			fn = keyTable.RawGetString("action")
		}
	})

	c.Config.State.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, luar.New(c.Config.State, c), luar.New(c.Config.State, e))
	// Returned value
	ret, ok := c.Config.State.Get(-1).(*lua.LUserData)
	if !ok {
		return e
	}

	// Remove returned value
	c.Config.State.Pop(1)
	return ret.Value.(*tcell.EventKey)
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
