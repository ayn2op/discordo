package main

import (
	"context"
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
	config         *util.Config
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
	config = util.NewConfig()

	app = ui.NewApp(onAppInputCapture)
	guildsTreeView = ui.NewGuildsTreeView(onGuildsTreeViewSelected, config.Theme)
	messagesTextView = ui.NewMessagesTextView(app, config.Theme)
	messageInputField = ui.NewMessageInputField(onMessageInputFieldInputCapture, config.Theme)
	mainFlex = ui.NewMainFlex(guildsTreeView, messagesTextView, messageInputField)

	token := util.GetItem(kr, "token")
	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsTreeView)

		discordSession = newSession("", "", token)
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

func onMessageInputFieldInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Key() {
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

	return event
}

func newSession(email string, password string, token string) *session.Session {
	api.UserAgent = "" +
		"Mozilla/5.0 (X11; Linux x86_64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/91.0.4472.164 Safari/537.36"
	gateway.DefaultIdentity.Browser = "Chrome"
	gateway.DefaultIdentity.OS = "Linux"
	gateway.DefaultIdentity.Device = ""

	var sess *session.Session
	var err error
	if email != "" && password != "" {
		sess, err = session.Login(email, password, "")
	} else if token != "" {
		sess, err = session.New(token)
	}

	if err != nil {
		panic(err)
	}

	sess.AddHandler(onSessionReady)
	sess.AddHandler(onSessionMessageCreate)
	if err = sess.Open(context.Background()); err != nil {
		panic(err)
	}

	return sess
}

func onSessionMessageCreate(m *gateway.MessageCreateEvent) {
	if currentChannel.ID == m.ChannelID {
		util.WriteMessage(messagesTextView, clientID, m.Message)
	}
}

func onSessionReady(r *gateway.ReadyEvent) {
	clientID = r.User.ID

	for _, g := range r.Guilds {
		gNode := tview.NewTreeNode(g.Name).
			SetReference(g).
			Collapse()
		guildsTreeView.GetRoot().AddChild(gNode)

		sort.Slice(g.Channels, func(i, j int) bool {
			return g.Channels[i].Position < g.Channels[j].Position
		})

		for _, c := range g.Channels {
			switch c.Type {
			case discord.GuildCategory:
				cNode := tview.NewTreeNode(c.Name).
					SetReference(c)
				gNode.AddChild(cNode)
			case discord.GuildText, discord.GuildNews:
				if c.ParentID == 0 || c.ParentID == discord.NullChannelID {
					cNode := tview.NewTreeNode("[::d]#" + c.Name + "[-:-:-]").
						SetReference(c)
					gNode.AddChild(cNode)
				}
			}
		}
	}
}

func onGuildsTreeViewSelected(n *tview.TreeNode) {
	switch ref := n.GetReference().(type) {
	case gateway.GuildCreateEvent:
		currentGuild = ref

		n.SetExpanded(!n.IsExpanded())
	case discord.Channel:
		switch ref.Type {
		case discord.GuildCategory:
			if len(n.GetChildren()) == 0 {
				for _, c := range currentGuild.Channels {
					if (c.Type == discord.GuildText || c.Type == discord.GuildNews) && c.ParentID == ref.ID {
						cNode := tview.NewTreeNode("[::d]#" + c.Name + "[-:-:-]").
							SetReference(c)
						n.AddChild(cNode)
					}
				}
			} else {
				n.SetExpanded(!n.IsExpanded())
			}
		case discord.GuildText, discord.GuildNews:
			if len(n.GetChildren()) == 0 {
				currentChannel = ref

				app.SetFocus(messageInputField)
				messagesTextView.Clear()
				messagesTextView.SetTitle(ref.Name)

				go func() {
					for _, t := range currentGuild.Threads {
						if t.ParentID == currentChannel.ID {
							cNode := tview.NewTreeNode("[::d]🗨 " + t.Name + "[::-]").
								SetReference(t)
							n.AddChild(cNode)
						}
					}
				}()

				go func() {
					msgs, _ := discordSession.Messages(ref.ID, config.GetMessagesLimit)
					for _, m := range msgs {
						util.WriteMessage(messagesTextView, clientID, m)
					}
				}()
			} else {
				n.SetExpanded(!n.IsExpanded())
			}
		case discord.GuildNewsThread, discord.GuildPrivateThread, discord.GuildPublicThread:
			currentChannel = ref

			app.SetFocus(messageInputField)
			messagesTextView.Clear()
			messagesTextView.SetTitle(ref.Name)

			go func() {
				msgs, _ := discordSession.Messages(ref.ID, config.GetMessagesLimit)
				for _, m := range msgs {
					util.WriteMessage(messagesTextView, clientID, m)
				}
			}()
		}
	}
}

func onLoginFormLoginButtonSelected() {
	email := loginForm.GetFormItem(0).(*tview.InputField).GetText()
	password := loginForm.GetFormItem(1).(*tview.InputField).GetText()
	if email == "" || password == "" {
		return
	}

	app.
		SetRoot(mainFlex, true).
		SetFocus(guildsTreeView)

	discordSession = newSession(email, password, "")

	go util.SetItem(kr, "token", discordSession.Token)
}
