package main

import (
	"os"
	"sort"
	"strings"

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
	guildsList        *tview.List
	channelsTreeView  *tview.TreeView
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex

	conf           *util.Config
	discordSession *session.Session
	clientID       discord.UserID
	guilds         []gateway.GuildCreateEvent
	guild          gateway.GuildCreateEvent
	channel        discord.Channel
)

func main() {
	conf = util.NewConfig()

	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor(conf.Theme.Background)

	if !conf.Theme.Borders {
		tview.Borders = struct {
			Horizontal       rune
			Vertical         rune
			TopLeft          rune
			TopRight         rune
			BottomLeft       rune
			BottomRight      rune
			LeftT            rune
			RightT           rune
			TopT             rune
			BottomT          rune
			Cross            rune
			HorizontalFocus  rune
			VerticalFocus    rune
			TopLeftFocus     rune
			TopRightFocus    rune
			BottomLeftFocus  rune
			BottomRightFocus rune
		}{}
	}

	app = ui.NewApp(onAppInputCapture)
	guildsList = ui.NewGuildsList(onGuildsListSelected, conf.Theme)
	channelsTreeView = ui.NewChannelsTreeView(onChannelsTreeViewSelected)
	messagesTextView = ui.NewMessagesTextView(app)
	messageInputField = ui.NewMessageInputField(onMessageInputFieldInputCapture, conf.Theme)
	mainFlex = ui.NewMainFlex(guildsList, channelsTreeView, messagesTextView, messageInputField)

	token := os.Getenv("DISCORDO_TOKEN")
	if t := util.GetPassword("token"); t != "" {
		token = t
	}

	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsList)

		discordSession = newSession(token)
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
		app.SetFocus(guildsList)
	case "Alt+Rune[2]":
		app.SetFocus(channelsTreeView)
	case "Alt+Rune[3]":
		app.SetFocus(messagesTextView)
	case "Alt+Rune[4]":
		app.SetFocus(messageInputField)
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

		discordSession.SendMessage(channel.ID, t)
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

func onSessionReady(r *gateway.ReadyEvent) {
	clientID = r.User.ID
	guilds = r.Guilds
	for _, g := range r.Guilds {
		guildsList.AddItem(g.Name, "", 0, nil)
	}
}

func onSessionMessageCreate(m *gateway.MessageCreateEvent) {
	if channel.ID == m.ChannelID {
		util.WriteMessage(messagesTextView, clientID, m.Message)
	}
}

func onGuildsListSelected(i int, _ string, _ string, _ rune) {
	app.SetFocus(channelsTreeView)
	messagesTextView.SetTitle("")
	messagesTextView.Clear()

	n := channelsTreeView.GetRoot()
	n.ClearChildren()

	guild = guilds[i]
	sort.SliceStable(guild.Channels, func(i, j int) bool {
		return guild.Channels[i].Position < guild.Channels[j].Position
	})

	for _, c := range guild.Channels {
		switch c.Type {
		case discord.GuildCategory:
			cn := tview.NewTreeNode(c.Name).
				SetReference(c)
			n.AddChild(cn)
		case discord.GuildText, discord.GuildNews:
			if c.ParentID == 0 || c.ParentID == discord.NullChannelID {
				cn := tview.NewTreeNode("[::d]#" + c.Name + "[::-]").
					SetReference(c)
				n.AddChild(cn)
			}
		case discord.GuildStageVoice, discord.GuildVoice:
			if c.ParentID == 0 || c.ParentID == discord.NullChannelID {
				cn := tview.NewTreeNode("[::d]🔊" + c.Name + "[::-]").
					SetReference(c)
				n.AddChild(cn)
			}
		}
	}
}

func onChannelsTreeViewSelected(n *tview.TreeNode) {
	r := n.GetReference().(discord.Channel)
	switch r.Type {
	case discord.GuildCategory:
		if len(n.GetChildren()) == 0 {
			for _, c := range guild.Channels {
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
			channel = r
			app.SetFocus(messageInputField)
			messagesTextView.Clear()

			title := "#" + r.Name
			if r.Topic != "" {
				title += " - " + r.Topic
			}
			messagesTextView.SetTitle(title)

			for _, t := range guild.Threads {
				if t.ParentID == channel.ID {
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
		channel = r
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
			SetRoot(guildsList, true).
			SetFocus(channelsTreeView)

		discordSession = newSession(l.Token)
		go util.SetPassword("token", l.Token)
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
				SetFocus(guildsList)

			discordSession = newSession(l.Token)
			go util.SetPassword("token", l.Token)
		})

		app.SetRoot(loginForm, true)
	}
}
