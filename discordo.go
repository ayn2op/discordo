package main

import (
	"sort"
	"strings"

	"github.com/99designs/keyring"
	"github.com/atotto/clipboard"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/ui"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

var (
	app               *tview.Application
	loginForm         *tview.Form
	guildsTreeView    *tview.TreeView
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex

	kr             keyring.Keyring
	conf           *util.Config
	discordSession *session.Session
	clientID       discord.UserID
	currentGuild   gateway.GuildCreateEvent
	currentChannel discord.Channel
)

func main() {
	tview.Borders.HorizontalFocus = tview.Borders.Horizontal
	tview.Borders.VerticalFocus = tview.Borders.Vertical
	tview.Borders.TopLeftFocus = tview.Borders.TopLeft
	tview.Borders.TopRightFocus = tview.Borders.TopRight
	tview.Borders.BottomLeftFocus = tview.Borders.BottomLeft
	tview.Borders.BottomRightFocus = tview.Borders.BottomRight
	tview.Borders.Horizontal = ' '
	tview.Borders.Vertical = ' '
	tview.Borders.TopLeft = ' '
	tview.Borders.TopRight = ' '
	tview.Borders.BottomLeft = ' '
	tview.Borders.BottomRight = ' '

	kr = util.OpenKeyringBackend()
	conf = util.NewConfig()

	app = ui.NewApp(onAppInputCapture)
	guildsTreeView = ui.NewGuildsTreeView(onGuildsTreeViewSelected, conf.Theme)
	messagesTextView = ui.NewMessagesTextView(app, conf.Theme)
	messageInputField = ui.NewMessageInputField(onMessageInputFieldInputCapture, conf.Theme)
	mainFlex = ui.NewMainFlex(guildsTreeView, messagesTextView, messageInputField)

	if t := util.GetItem(kr, "token"); t != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

		discordSession = newSession(t)
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
	case "Alt+Rune[1]":
		app.SetFocus(guildsTreeView)
	case "Alt+Rune[2]":
		app.SetFocus(messagesTextView)
	case "Alt+Rune[3]":
		if messageInputField != nil {
			app.SetFocus(messageInputField)
		}
	}

	return e
}

func onMessageInputFieldInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyEnter:
		t := strings.TrimSpace(messageInputField.GetText())
		if t == "" {
			return nil
		}

		discordSession.SendMessage(currentChannel.ID, t)
		messageInputField.SetText("")
	case tcell.KeyCtrlV:
		text, _ := clipboard.ReadAll()
		text = messageInputField.GetText() + text
		messageInputField.SetText(text)
	}

	return e
}

func newSession(token string) (s *session.Session) {
	api.UserAgent = "" +
		"Mozilla/5.0 (X11; Linux x86_64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/92.0.4515.131 Safari/537.36"
	gateway.DefaultIdentity.Browser = "Chrome"
	gateway.DefaultIdentity.OS = "Linux"
	gateway.DefaultIdentity.Device = ""

	var err error
	s, err = session.New(token)
	if err != nil {
		panic(err)
	}

	s.AddHandler(onSessionReady)
	s.AddHandler(onSessionMessageCreate)
	if err = s.Open(s.Context()); err != nil {
		panic(err)
	}

	return
}

func onSessionMessageCreate(m *gateway.MessageCreateEvent) {
	if currentChannel.ID == m.ChannelID {
		util.WriteMessage(messagesTextView, clientID, m.Message)
	}
}

func onSessionReady(r *gateway.ReadyEvent) {
	clientID = r.User.ID

	for _, g := range r.Guilds {
		gn := tview.NewTreeNode(g.Name).
			SetReference(g).
			Collapse()
		guildsTreeView.GetRoot().AddChild(gn)

		sort.Slice(g.Channels, func(i, j int) bool {
			return g.Channels[i].Position < g.Channels[j].Position
		})

		for _, c := range g.Channels {
			switch c.Type {
			case discord.GuildCategory:
				cn := tview.NewTreeNode(c.Name).
					SetReference(c)
				gn.AddChild(cn)
			case discord.GuildText, discord.GuildNews:
				if c.ParentID == 0 || c.ParentID == discord.NullChannelID {
					cn := tview.NewTreeNode("[::d]#" + c.Name + "[::-]").
						SetReference(c)
					gn.AddChild(cn)
				}
			case discord.GuildStageVoice, discord.GuildVoice:
				if c.ParentID == 0 || c.ParentID == discord.NullChannelID {
					cn := tview.NewTreeNode("[::d]🔊" + c.Name + "[::-]").
						SetReference(c)
					gn.AddChild(cn)
				}
			}
		}
	}
}

func onGuildsTreeViewSelected(n *tview.TreeNode) {
	switch r := n.GetReference().(type) {
	case gateway.GuildCreateEvent:
		currentGuild = r
		n.SetExpanded(!n.IsExpanded())
	case discord.Channel:
		switch r.Type {
		case discord.GuildCategory:
			if len(n.GetChildren()) == 0 {
				for _, c := range currentGuild.Channels {
					switch c.Type {
					case discord.GuildText, discord.GuildNews:
						if c.ParentID == r.ID {
							cn := tview.NewTreeNode("[::d]#" + c.Name + "[::-]").
								SetReference(c)
							n.AddChild(cn)
						}
					case discord.GuildStageVoice, discord.GuildVoice:
						if c.ParentID == r.ID {
							cn := tview.NewTreeNode("[::d]🔊" + c.Name + "[::-]").
								SetReference(c)
							n.AddChild(cn)
						}
					}
				}
			} else {
				n.SetExpanded(!n.IsExpanded())
			}
		case discord.GuildText, discord.GuildNews:
			if len(n.GetChildren()) == 0 {
				currentChannel = r
				app.SetFocus(messageInputField)
				messagesTextView.Clear()

				title := "#" + r.Name
				if r.Topic != "" {
					title += " - " + r.Topic
				}
				messagesTextView.SetTitle(title)

				for _, t := range currentGuild.Threads {
					if t.ParentID == currentChannel.ID {
						cn := tview.NewTreeNode("[::d]🗨 " + t.Name + "[::-]").
							SetReference(t)
						n.AddChild(cn)
					}
				}

				go func() {
					msgs, _ := discordSession.Messages(r.ID, conf.GetMessagesLimit)
					for i := len(msgs) - 1; i >= 0; i-- {
						util.WriteMessage(messagesTextView, clientID, msgs[i])
					}
				}()
			} else {
				n.SetExpanded(!n.IsExpanded())
			}
		case discord.GuildNewsThread, discord.GuildPrivateThread, discord.GuildPublicThread:
			currentChannel = r
			app.SetFocus(messageInputField)
			messagesTextView.Clear()
			messagesTextView.SetTitle(r.Name)

			go func() {
				msgs, _ := discordSession.Messages(r.ID, conf.GetMessagesLimit)
				for i := len(msgs) - 1; i >= 0; i-- {
					util.WriteMessage(messagesTextView, clientID, msgs[i])
				}
			}()
		case discord.GuildStageVoice, discord.GuildVoice:
			messagesTextView.Clear()
			messagesTextView.SetTitle(r.Name)
		}
	}
}

func onLoginFormLoginButtonSelected() {
	email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
	password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	// Make a scratch HTTP client without a token
	client := api.NewClient("")
	// Try to login without TOTP
	l, err := client.Login(email, password)
	if err != nil {
		panic(err)
	}

	if l.Token != "" && !l.MFA {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

		discordSession = newSession(l.Token)
		go util.SetItem(kr, "token", l.Token)
	} else if l.MFA {
		loginForm = ui.NewMfaLoginForm(func() {
			code := loginForm.GetFormItem(0).(*tview.InputField).GetText()
			if code == "" {
				return
			}

			l, err := client.TOTP(code, l.Ticket)
			if err != nil {
				panic(err)
			}

			app.
				SetRoot(mainFlex, true).
				SetFocus(guildsTreeView)

			discordSession = newSession(l.Token)
			go util.SetItem(kr, "token", l.Token)
		})

		app.SetRoot(loginForm, true)
	}
}
