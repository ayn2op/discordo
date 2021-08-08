package main

import (
	"context"
	"sort"
	"strings"

	"github.com/99designs/keyring"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/diamondburned/arikawa/v3/session"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/state/store/defaultstore"
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/ui"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

var (
	app               *tview.Application
	loginModal        *tview.Modal
	loginForm         *tview.Form
	guildsDropDown    *tview.DropDown
	channelsTreeView  *tview.TreeView
	channelsTreeNode  *tview.TreeNode
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex

	loginVia       string
	kr             keyring.Keyring
	config         *util.Config
	discordSession *session.Session
	discordState   *state.State
	currentGuild   discord.Guild
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

	loginModal = ui.NewLoginModal(onLoginModalDone)
	guildsDropDown = ui.NewGuildsDropDown(onGuildsDropDownSelected, config.Theme)
	channelsTreeNode = tview.NewTreeNode("")
	channelsTreeView = ui.NewChannelsTreeView(channelsTreeNode, onChannelsTreeViewSelected, config.Theme)
	messagesTextView = ui.NewMessagesTextView(onMessagesTextViewChanged, config.Theme)
	mainFlex = ui.NewMainFlex(guildsDropDown, channelsTreeView, messagesTextView)
	app = ui.NewApp(onAppInputCapture)

	token := util.GetItem(kr, "token")
	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsDropDown)

		discordSession = newSession("", "", token)
	} else {
		app.SetRoot(loginModal, true)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case "Ctrl+G":
		app.SetFocus(guildsDropDown)
	case "Ctrl+J":
		app.SetFocus(channelsTreeView)
	case "Ctrl+K":
		app.SetFocus(messagesTextView)
	case "Ctrl+L":
		if messageInputField != nil {
			app.SetFocus(messageInputField)
		}
	}

	return e
}

func onMessagesTextViewChanged() {
	app.Draw()
}

func onLoginModalDone(i int, label string) {
	if label == ui.LoginViaEmailPasswordLoginModalButton {
		loginVia = "emailpassword"
		loginForm = ui.NewLoginForm(loginVia, onLoginFormLoginButtonSelected)
		app.SetRoot(loginForm, true)
	} else if label == ui.LoginViaTokenLoginModalButton {
		loginVia = "token"
		loginForm = ui.NewLoginForm(loginVia, onLoginFormLoginButtonSelected)
		app.SetRoot(loginForm, true)
	}
}

func newSession(email string, password string, token string) *session.Session {
	api.UserAgent = "" +
		"Mozilla/5.0 (X11; Linux x86_64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/91.0.4472.164 Safari/537.36"
	gateway.DefaultIdentity.Browser = "Chrome"
	gateway.DefaultIdentity.OS = "Linux"

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

	discordState = state.NewFromSession(sess, defaultstore.New())

	sess.AddHandler(func(r *gateway.ReadyEvent) {
		for i := range r.Guilds {
			guildsDropDown.AddOption(r.Guilds[i].Name, nil)
		}
	})
	sess.AddHandler(func(g *gateway.GuildCreateEvent) {
		guildsDropDown.AddOption(g.Name, nil)
	})
	sess.AddHandler(func(m *gateway.MessageCreateEvent) {
		if currentChannel.ID == m.ChannelID {
			util.WriteMessage(messagesTextView, discordState, m.Message)
		}
	})
	if err = sess.Open(context.Background()); err != nil {
		panic(err)
	}

	return sess
}

func onGuildsDropDownSelected(_ string, i int) {
	channelsTreeView.SetTitle("Channels")
	channelsTreeNode.ClearChildren()
	messagesTextView.
		Clear().
		SetTitle("")
	app.SetFocus(channelsTreeView)

	if messageInputField != nil {
		mainFlex.RemoveItem(messageInputField)
		messageInputField = nil
	}

	currentGuild = discordState.Ready().Guilds[i].Guild
	channels, _ := discordState.Cabinet.Channels(currentGuild.ID)
	sort.SliceStable(channels, func(i, j int) bool {
		return channels[i].Position < channels[j].Position
	})

	for i := range channels {
		channel := channels[i]
		channelNode := tview.NewTreeNode(channel.Name).
			SetReference(channel)
		switch channel.Type {
		case discord.GuildCategory:
			channelsTreeNode.AddChild(channelNode)
		case discord.GuildText, discord.GuildNews:
			if channel.CategoryID == 0 || channel.CategoryID == discord.NullChannelID {
				channelNode.SetText("[::d]#" + channel.Name + "[-:-:-]")
				channelsTreeNode.AddChild(channelNode)
			}
		}
	}
}

func onChannelsTreeViewSelected(n *tview.TreeNode) {
	currentChannel = n.GetReference().(discord.Channel)
	switch currentChannel.Type {
	case discord.GuildCategory:
		if len(n.GetChildren()) == 0 {
			channels, _ := discordState.Cabinet.Channels(currentGuild.ID)
			sort.SliceStable(channels, func(i, j int) bool {
				return channels[i].Position < channels[j].Position
			})

			for i := range channels {
				channel := channels[i]
				if (channel.Type == discord.GuildText || channel.Type == discord.GuildNews) && channel.CategoryID == currentChannel.ID {
					channelNode := tview.NewTreeNode("[::d]#" + channel.Name + "[-:-:-]").
						SetReference(channel)
					n.AddChild(channelNode)
				}
			}
		} else {
			n.SetExpanded(!n.IsExpanded())
		}
	case discord.GuildText, discord.GuildNews:
		if messageInputField == nil {
			messageInputField = ui.NewMessageInputField(onMessageInputFieldDone, config.Theme)
			mainFlex.AddItem(messageInputField, 3, 1, false)
		}
		app.SetFocus(messageInputField)

		messagesTextView.Clear()
		messagesTextView.SetTitle(currentChannel.Name)

		messages, _ := discordSession.Messages(currentChannel.ID, config.GetMessagesLimit)
		for i := len(messages) - 1; i >= 0; i-- {
			util.WriteMessage(messagesTextView, discordState, messages[i])
		}
	}
}

func onMessageInputFieldDone(k tcell.Key) {
	if k == tcell.KeyEnter {
		currentText := strings.TrimSpace(messageInputField.GetText())
		if currentText == "" {
			return
		}

		messageInputField.SetText("")
		discordSession.SendMessage(currentChannel.ID, currentText)
	}
}

func onLoginFormLoginButtonSelected() {
	if loginVia == "emailpassword" {
		email := loginForm.GetFormItemByLabel("Email").(*tview.InputField).GetText()
		password := loginForm.GetFormItemByLabel("Password").(*tview.InputField).GetText()
		if email == "" || password == "" {
			return
		}

		discordSession = newSession(email, password, "")

		go util.SetItem(kr, "token", discordSession.Token)
	} else if loginVia == "token" {
		token := loginForm.GetFormItemByLabel("Token").(*tview.InputField).GetText()
		if token == "" {
			return
		}

		discordSession = newSession("", "", token)

		go util.SetItem(kr, "token", token)
	}

	app.
		SetRoot(mainFlex, true).
		SetFocus(guildsDropDown)
}
