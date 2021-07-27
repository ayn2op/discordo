package main

import (
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/gdamore/tcell/v2"
	"github.com/rigormorrtiss/discordo/ui"
	"github.com/rigormorrtiss/discordo/util"
	"github.com/rivo/tview"
)

var app *tview.Application
var loginModal *tview.Modal
var loginForm *tview.Form
var guildsDropDown *tview.DropDown
var channelsList *tview.List
var messagesTextView *tview.TextView
var messageInputField *tview.InputField
var mainFlex *tview.Flex

var loginVia string
var session *discordgo.Session
var currentGuild *discordgo.Guild
var currentChannel *discordgo.Channel

func main() {
	tview.Styles.PrimitiveBackgroundColor = tcell.GetColor("#1C1E26")

	loginModal = ui.NewLoginModal(onLoginModalDone)
	guildsDropDown = ui.NewGuildsDropDown(onGuildsDropDownSelected)
	channelsList = ui.NewChannelsList(onChannelsListSelected)
	messagesTextView = ui.NewMessagesTextView(onMessagesTextViewChanged)
	mainFlex = ui.NewMainFlex(guildsDropDown, channelsList, messagesTextView)
	app = ui.NewApplication()

	token := util.GetPassword("token")
	if token != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsDropDown)

		session = newSession("", "", token)
	} else {
		app.SetRoot(loginModal, true)
	}

	if err := app.EnableMouse(true).Run(); err != nil {
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
	var sess *discordgo.Session
	var err error
	if email != "" && password != "" {
		sess, err = discordgo.New(email, password)
		if err != nil {
			panic(err)
		}

		sess.AddHandler(onReady)
	} else if token != "" {
		sess, err = discordgo.New(token)
		if err != nil {
			panic(err)
		}

		if !strings.HasPrefix(token, "Bot ") {
			sess.AddHandler(onReady)
		}
	}

	sess.AddHandler(onGuildCreate)
	sess.AddHandler(onMessageCreate)

	sess.Identify.Intents = discordgo.IntentsAllWithoutPrivileged
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
	if currentChannel.ID == message.ChannelID {
		util.WriteMessage(messagesTextView, session, message.Message)
	}
}

func onGuildsDropDownSelected(text string, _ int) {
	channelsList.Clear()
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

	for i := range currentGuild.Channels {
		channel := currentGuild.Channels[i]
		channelsList.AddItem(channel.Name, "", 0, nil)
	}

	app.SetFocus(channelsList)
}

func onChannelsListSelected(i int, mainText string, secondaryText string, _ rune) {
	messagesTextView.Clear()

	if messageInputField == nil {
		messageInputField = ui.NewMessageInputField(onMessageInputFieldDone)
		mainFlex.AddItem(messageInputField, 3, 1, false)
	}

	app.SetFocus(messageInputField)

	currentChannel = currentGuild.Channels[i]

	messagesTextView.SetTitle(currentChannel.Name)

	messages := util.GetMessages(session, currentChannel.ID, 50)
	for i := len(messages) - 1; i >= 0; i-- {
		util.WriteMessage(messagesTextView, session, messages[i])
	}
}

func onMessageInputFieldDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		currentText := messageInputField.GetText()
		currentText = strings.TrimSpace(currentText)

		if currentText == "" {
			return
		}

		util.SendMessage(session, currentChannel.ID, currentText)

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
