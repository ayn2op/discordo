package main

import (
	"sort"
	"strings"

	"github.com/bwmarrin/discordgo"
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
	config         *util.Config
	session        *discordgo.Session
	currentGuild   *discordgo.Guild
	currentChannel *discordgo.Channel
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

	config = util.NewConfig()
	loginModal = ui.NewLoginModal(onLoginModalDone)
	guildsDropDown = ui.NewGuildsDropDown(onGuildsDropDownSelected, config.Theme)
	channelsTreeNode = ui.NewChannelsTreeNode()
	channelsTreeView = ui.NewChannelsTreeView(channelsTreeNode, onChannelsTreeViewSelected, config.Theme)
	messagesTextView = ui.NewMessagesTextView(onMessagesTextViewChanged, config.Theme)
	mainFlex = ui.NewMainFlex(guildsDropDown, channelsTreeView, messagesTextView)
	app = ui.NewApp()

	token := util.GetPassword("token")
	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsDropDown)

		session = newSession("", "", token)
	} else {
		app.SetRoot(loginModal, true)
	}

	if err := app.Run(); err != nil {
		panic(err)
	}
}

func onLoginFormQuitButtonSelected() {
	app.Stop()
}

func onMessagesTextViewChanged() {
	app.Draw()
}

func onLoginModalDone(buttonIndex int, buttonLabel string) {
	if buttonLabel == ui.LoginViaEmailPasswordLoginModalButton {
		loginVia = "emailpassword"
		loginForm = ui.NewLoginForm(loginVia, onLoginFormLoginButtonSelected, onLoginFormQuitButtonSelected)
		app.SetRoot(loginForm, true)
	} else if buttonLabel == ui.LoginViaTokenLoginModalButton {
		loginVia = "token"
		loginForm = ui.NewLoginForm(loginVia, onLoginFormLoginButtonSelected, onLoginFormQuitButtonSelected)
		app.SetRoot(loginForm, true)
	}
}

func newSession(email string, password string, token string) *discordgo.Session {
	userAgent := "" +
		"Mozilla/5.0 (X11; Linux x86_64) " +
		"AppleWebKit/537.36 (KHTML, like Gecko) " +
		"Chrome/91.0.4472.164 Safari/537.36"

	var sess *discordgo.Session
	var err error
	if email != "" && password != "" {
		sess, err = discordgo.New(email, password)
		if err != nil {
			panic(err)
		}

		sess.UserAgent = userAgent
		sess.Identify.Properties.Browser = "Chrome"
		sess.Identify.Properties.OS = "Linux"

		sess.AddHandler(onReady)
	} else if token != "" {
		sess, err = discordgo.New(token)
		if err != nil {
			panic(err)
		}

		if !strings.HasPrefix(token, "Bot ") {
			sess.UserAgent = userAgent
			sess.Identify.Properties.Browser = "Chrome"
			sess.Identify.Properties.OS = "Linux"

			sess.AddHandler(onReady)
		}
	}

	sess.AddHandler(onGuildCreate)
	sess.AddHandler(onMessageCreate)
	if err = sess.Open(); err != nil {
		panic(err)
	}

	return sess
}

func onGuildCreate(_ *discordgo.Session, guild *discordgo.GuildCreate) {
	guildsDropDown.AddOption(guild.Name, nil)
}

func onReady(_ *discordgo.Session, ready *discordgo.Ready) {
	for i := range ready.Guilds {
		guildsDropDown.AddOption(ready.Guilds[i].Name, nil)
	}
}

func onMessageCreate(_ *discordgo.Session, message *discordgo.MessageCreate) {
	if currentChannel != nil && currentChannel.ID == message.ChannelID {
		util.WriteMessage(messagesTextView, session, message.Message)
	}
}

func onGuildsDropDownSelected(text string, _ int) {
	channelsTreeNode.ClearChildren()
	messagesTextView.Clear()

	if messageInputField != nil {
		mainFlex.RemoveItem(messageInputField)
		messageInputField = nil
	}

	guilds := session.State.Guilds
	for i := range guilds {
		guild := guilds[i]
		if guild.Name == text {
			currentGuild = guild
			break
		}
	}

	channelsTreeView.SetTitle("Channels")
	app.SetFocus(channelsTreeView)

	channels := currentGuild.Channels
	sort.Slice(channels, func(i, j int) bool {
		return channels[i].Position < channels[j].Position
	})

	for i := range channels {
		channel := channels[i]
		channelNode := tview.NewTreeNode(channel.Name).
			SetReference(channel)

		switch channel.Type {
		case discordgo.ChannelTypeGuildCategory:
			channelNode.SetColor(tcell.GetColor(config.Theme.TreeNodeForeground))
			channelsTreeNode.AddChild(channelNode)
		default:
			if channel.ParentID == "" {
				channelNode.SetText("[::d]#" + channel.Name + "[-:-:-]")
				channelsTreeNode.AddChild(channelNode)
			}
		}
	}
}

func onChannelsTreeViewSelected(node *tview.TreeNode) {
	messagesTextView.Clear()

	if messageInputField == nil {
		messageInputField = ui.NewMessageInputField(onMessageInputFieldDone, config.Theme)
		mainFlex.AddItem(messageInputField, 3, 1, false)
	}

	currentChannel = node.GetReference().(*discordgo.Channel)
	switch currentChannel.Type {
	case discordgo.ChannelTypeGuildCategory:
		if len(node.GetChildren()) == 0 {
			for i := range currentGuild.Channels {
				channel := currentGuild.Channels[i]
				if channel.ParentID == currentChannel.ID {
					channelNode := tview.NewTreeNode("[::d]#" + channel.Name + "[-:-:-]").
						SetReference(channel)
					node.AddChild(channelNode)
				}
			}
		} else {
			node.SetExpanded(!node.IsExpanded())
		}
	case discordgo.ChannelTypeGuildText:
		messagesTextView.SetTitle(currentChannel.Name)
		app.SetFocus(messageInputField)

		messages, err := session.ChannelMessages(currentChannel.ID, config.GetMessagesLimit, "", "", "")
		if err != nil {
			panic(err)
		}

		for i := len(messages) - 1; i >= 0; i-- {
			util.WriteMessage(messagesTextView, session, messages[i])
		}
	}
}

func onMessageInputFieldDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		currentText := messageInputField.GetText()
		currentText = strings.TrimSpace(currentText)

		if currentText == "" {
			return
		}

		_, err := session.ChannelMessageSend(currentChannel.ID, currentText)
		if err != nil {
			panic(err)
		}

		messageInputField.SetText("")
	}
}

func onLoginFormLoginButtonSelected() {
	if loginVia == "emailpassword" {
		email := loginForm.GetFormItemByLabel("Email").(*tview.InputField).GetText()
		password := loginForm.GetFormItemByLabel("Password").(*tview.InputField).GetText()
		if email == "" || password == "" {
			return
		}

		session = newSession(email, password, "")
		util.SetPassword("token", session.Token)
	} else if loginVia == "token" {
		token := loginForm.GetFormItemByLabel("Token").(*tview.InputField).GetText()
		if token == "" {
			return
		}

		session = newSession("", "", token)
		util.SetPassword("token", token)
	}

	app.
		SetRoot(mainFlex, true).
		SetFocus(guildsDropDown)
}
