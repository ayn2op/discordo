package ui

import (
	"context"
	_ "embed"
	"os"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

//go:embed config.lua
var cfg []byte

// Core initializes the application, UI elements, configuration, session, and state. It also manages the application, session, and state.
type Core struct {
	Application   *tview.Application
	MainFlex      *tview.Flex
	GuildsTree    *GuildsTree
	ChannelsTree  *ChannelsTree
	MessagesPanel *MessagesPanel
	MessageInput  *MessageInput

	State  *state.State
	LState *lua.LState

	token string
	cfg   string
}

func NewCore(token string, cfg string) *Core {
	c := &Core{
		Application: tview.NewApplication(),
		MainFlex:    tview.NewFlex(),
		LState:      lua.NewState(),

		token: token,
		cfg:   cfg,
	}

	c.MainFlex.SetInputCapture(c.onInputCapture)
	c.GuildsTree = NewGuildsTree(c)
	c.ChannelsTree = NewChannelsTree(c)
	c.MessagesPanel = NewMessagesPanel(c)
	c.MessageInput = NewMessageInput(c)
	return c
}

func (c *Core) Run() error {
	err := c.loadConfig(c.cfg)
	if err != nil {
		return err
	}

	c.register()
	err = c.LState.DoFile(c.cfg)
	if err != nil {
		return err
	}

	c.Application.EnableMouse(lua.LVAsBool(c.LState.GetGlobal("mouse")))

	identifyProperties, ok := c.LState.GetGlobal("identifyProperties").(*lua.LTable)
	if !ok {
		identifyProperties = c.LState.NewTable()
	}

	userAgent := lua.LVAsString(identifyProperties.RawGetString("userAgent"))

	c.State = state.NewWithIdentifier(gateway.NewIdentifier(gateway.IdentifyCommand{
		Token:   c.token,
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

func (c *Core) register() {
	c.LState.SetGlobal("key", c.LState.NewFunction(c.keyLua))
	// Messages panel
	c.LState.SetGlobal("openMessageActionsList", c.LState.NewFunction(c.MessagesPanel.openMessageActionsListLua))
	c.LState.SetGlobal("selectPreviousMessage", c.LState.NewFunction(c.MessagesPanel.selectPreviousMessageLua))
	c.LState.SetGlobal("selectNextMessage", c.LState.NewFunction(c.MessagesPanel.selectNextMessageLua))
	c.LState.SetGlobal("selectFirstMessage", c.LState.NewFunction(c.MessagesPanel.selectFirstMessageLua))
	c.LState.SetGlobal("selectLastMessage", c.LState.NewFunction(c.MessagesPanel.selectLastMessageLua))
	// Message input
	c.LState.SetGlobal("openExternalEditor", c.LState.NewFunction(c.MessageInput.openExternalEditorLua))
	c.LState.SetGlobal("pasteClipboardContent", c.LState.NewFunction(c.MessageInput.pasteClipboardContentLua))
}

func (c *Core) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if c.MessageInput.HasFocus() {
		return e
	}
	// If the main flex is nil, that is, it is not initialized yet, then the login form is currently focused.
	if c.MainFlex == nil {
		return e
	}

	keysTable, ok := c.LState.GetGlobal("keys").(*lua.LTable)
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

	c.LState.CallByParam(lua.P{
		Fn:      fn,
		NRet:    1,
		Protect: true,
	}, luar.New(c.LState, c), luar.New(c.LState, e))
	// Returned value
	ret, ok := c.LState.Get(-1).(*lua.LUserData)
	if !ok {
		return e
	}

	// Remove returned value
	c.LState.Pop(1)
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

func (c *Core) loadConfig(path string) error {
	// Create a new configuration file if it does not exist already; otherwise, open the existing file with read-write flag.
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	// If the configuration file is empty, that is, its size is zero, write the default configuration to the file.
	if fi.Size() == 0 {
		f.Write(cfg)
		f.Sync()
	}

	return nil
}

func (c *Core) keyLua(s *lua.LState) int {
	keyTable := s.NewTable()
	keyTable.RawSetString("name", s.Get(1))
	keyTable.RawSetString("description", s.Get(2))
	keyTable.RawSetString("action", s.Get(3))

	s.Push(keyTable) // Push the result
	return 1         // Number of results
}
