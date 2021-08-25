package main

import (
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordgo"
	"github.com/rigormorrtiss/discordo/ui"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

var (
	app               *tview.Application
	loginForm         *tview.Form
	guildsTreeView    *tview.TreeView
	channelsTreeView  *tview.TreeView
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex

	config          *util.Config
	session         *discordgo.Session
	selectedChannel *discordgo.Channel
)

func main() {
	config = util.NewConfig()

	if config.Theme != nil {
		tview.Styles = *config.Theme
	}

	app = tview.NewApplication().
		EnableMouse(config.Mouse).
		SetInputCapture(onAppInputCapture)
	guildsTreeView = ui.NewGuildsTreeView(onGuildsTreeViewSelected)
	channelsTreeView = ui.NewChannelsTreeView(onChannelsTreeViewSelected)
	messagesTextView = ui.NewMessagesTextView(app)
	messageInputField = ui.NewMessageInputField(onMessageInputFieldInputCapture)
	mainFlex = ui.NewMainFlex(guildsTreeView, channelsTreeView, messagesTextView, messageInputField)

	token := config.Token
	if t := util.GetPassword("token"); t != "" {
		token = t
	}

	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

		session = newSession()
		session.Token = token
		session.Identify.Token = token
		if err := session.Open(); err != nil {
			panic(err)
		}
	} else {
		loginForm = ui.NewLoginForm(onLoginFormLoginButtonSelected)
		app.SetRoot(loginForm, true)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case "Alt+Rune[g]":
		app.SetFocus(guildsTreeView)
	case "Alt+Rune[c]":
		app.SetFocus(channelsTreeView)
	case "Alt+Rune[m]":
		app.SetFocus(messagesTextView)
	case "Alt+Rune[i]":
		app.SetFocus(messageInputField)
	}

	return e
}

func onMessageInputFieldInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyEnter:
		if selectedChannel != nil {
			text := strings.TrimSpace(messageInputField.GetText())
			if text == "" {
				return nil
			}

			session.ChannelMessageSend(selectedChannel.ID, text)
			messageInputField.SetText("")
		}
	case tcell.KeyCtrlV:
		text, _ := clipboard.ReadAll()
		text = messageInputField.GetText() + text
		messageInputField.SetText(text)
	}

	return e
}

func newSession() *discordgo.Session {
	s, err := discordgo.New()
	if err != nil {
		panic(err)
	}

	s.UserAgent = "" +
		"Mozilla/5.0 (X11; Linux x86_64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/92.0.4515.131 Safari/537.36"
	s.Identify.Compress = false
	s.Identify.Intents = 0
	s.Identify.LargeThreshold = 0
	s.Identify.Properties.Device = ""
	s.Identify.Properties.Browser = "Chrome"
	s.Identify.Properties.OS = "Linux"

	s.AddHandlerOnce(onSessionReady)
	s.AddHandler(onSessionMessageCreate)

	return s
}

func onSessionReady(_ *discordgo.Session, r *discordgo.Ready) {
	sort.Slice(r.Guilds, func(a, b int) bool {
		found := false
		for _, gID := range r.Settings.GuildPositions {
			if found {
				if gID == r.Guilds[b].ID {
					return true
				}
			} else {
				if gID == r.Guilds[a].ID {
					found = true
				}
			}
		}

		return false
	})

	rootN := guildsTreeView.GetRoot()
	for _, g := range r.Guilds {
		gn := tview.NewTreeNode(g.Name).
			SetReference(g.ID)
		rootN.AddChild(gn)
	}

	guildsTreeView.SetCurrentNode(rootN)
}

func onSessionMessageCreate(_ *discordgo.Session, m *discordgo.MessageCreate) {
	if selectedChannel != nil && selectedChannel.ID == m.ChannelID {
		util.WriteMessage(messagesTextView, m.Message, session.State.Ready.User.ID)
	}
}

func onGuildsTreeViewSelected(gn *tview.TreeNode) {
	selectedChannel = nil
	app.SetFocus(channelsTreeView)
	messagesTextView.
		Clear().
		SetTitle("")

	gID := gn.GetReference().(string)
	g, _ := session.State.Guild(gID)
	cs := g.Channels
	sort.Slice(cs, func(i, j int) bool {
		return cs[i].Position < cs[j].Position
	})

	rootN := channelsTreeView.GetRoot()
	rootN.ClearChildren()
	// Top-level channels
	ui.CreateTopLevelChannelsTreeNodes(session.State, rootN, cs)
	// Category channels
	ui.CreateCategoryChannelsTreeNodes(session.State, rootN, cs)
	// Second-level channels
	ui.CreateSecondLevelChannelsTreeNodes(session.State, channelsTreeView, rootN, cs)

	channelsTreeView.SetCurrentNode(rootN)
}

func onChannelsTreeViewSelected(n *tview.TreeNode) {
	cID := n.GetReference().(string)
	c, _ := session.State.Channel(cID)
	switch c.Type {
	case discordgo.ChannelTypeGuildCategory:
		n.SetExpanded(!n.IsExpanded())
	case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews:
		if len(n.GetChildren()) == 0 {
			selectedChannel = c
			app.SetFocus(messageInputField)

			title := "#" + c.Name
			if c.Topic != "" {
				title += " - " + c.Topic
			}
			messagesTextView.
				Clear().
				SetTitle(title)

			go writeMessages(c.ID)
		} else {
			n.SetExpanded(!n.IsExpanded())
		}
	}
}

func writeMessages(cID string) {
	msgs, _ := session.ChannelMessages(cID, config.GetMessagesLimit, "", "", "")
	for i := len(msgs) - 1; i >= 0; i-- {
		util.WriteMessage(messagesTextView, msgs[i], session.State.Ready.User.ID)
	}
}

func onLoginFormLoginButtonSelected() {
	email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
	password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	session = newSession()
	// Try to login without TOTP
	lr, err := util.Login(session, email, password)
	if err != nil {
		panic(err)
	}

	if lr.Token != "" && !lr.MFA {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

		session.Token = lr.Token
		session.Identify.Token = lr.Token
		if err = session.Open(); err != nil {
			panic(err)
		}

		go util.SetPassword("token", lr.Token)
	} else if lr.MFA {
		loginForm = ui.NewMfaLoginForm(func() {
			code := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			if code == "" {
				return
			}

			lr, err = util.TOTP(session, code, lr.Ticket)
			if err != nil {
				panic(err)
			}

			app.
				SetRoot(mainFlex, true).
				SetFocus(guildsTreeView)

			session.Token = lr.Token
			session.Identify.Token = lr.Token
			if err = session.Open(); err != nil {
				panic(err)
			}

			go util.SetPassword("token", lr.Token)
		})

		app.SetRoot(loginForm, true)
	}
}
