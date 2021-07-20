package main

import (
	"strings"

	"github.com/diamondburned/arikawa/v2/api"
	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/gateway"
	"github.com/diamondburned/arikawa/v2/session"
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
var discordSession *session.Session
var guilds []gateway.GuildCreateEvent
var currentGuild gateway.GuildCreateEvent
var currentChannel discord.Channel

func main() {
	loginModal = ui.NewLoginModal(onLoginModalDone)
	guildsDropDown = ui.NewGuildsDropDown(onGuildsDropDownSelected)
	channelsList = ui.NewChannelsList(onChannelsListSelected)
	messagesTextView = ui.NewMessagesTextView(onMessagesTextViewChanged)
	mainFlex = ui.NewMainFlex(guildsDropDown, channelsList, messagesTextView)
	app = ui.NewApplication(onApplicationInputCapture)

	email := util.GetPassword("email")
	password := util.GetPassword("password")
	token := util.GetPassword("token")
	if email != "" && password != "" {
		app.
			SetRoot(mainFlex, true).
			SetFocus(guildsDropDown)

		discordSession = newSession(email, password, "")
	} else if token != "" {
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

func onLoginFormQuitButtonSelected() {
	app.Stop()
}

func onApplicationInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if event.Key() == tcell.KeyCtrlR {
		app.Sync()
	}

	return event
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

func newSession(email string, password string, token string) *session.Session {
	var sess *session.Session
	var err error
	if email != "" && password != "" {
		api.UserAgent = `Mozilla/5.0 (X11; Linux x86_64; rv:90.0)` +
			`Gecko/20100101 Firefox/90.0`
		gateway.DefaultIdentity = gateway.IdentifyProperties{
			OS:      "Linux",
			Browser: "Firefox",
			Device:  "",
		}

		sess, err = session.Login(email, password, "")
		if err != nil {
			panic(err)
		}

		sess.AddHandler(onReady)
	} else if token != "" {
		sess, err = session.New(token)
		if err != nil {
			panic(err)
		}

		sess.AddHandler(onGuildCreate)
		sess.Gateway.AddIntents(gateway.IntentGuilds)
		sess.Gateway.AddIntents(gateway.IntentGuildMessages)
	}

	sess.AddHandler(onMessageCreate)
	if err = sess.Open(); err != nil {
		panic(err)
	}

	return sess
}

func onGuildCreate(guild *gateway.GuildCreateEvent) {
	guildsDropDown.AddOption(guild.Name, nil)
	guilds = append(guilds, *guild)
}

func onReady(ready *gateway.ReadyEvent) {
	guilds = ready.Guilds
	for i := 0; i < len(guilds); i++ {
		guildsDropDown.AddOption(guilds[i].Name, nil)
	}
}

func onMessageCreate(message *gateway.MessageCreateEvent) {
	if currentChannel.ID == message.ChannelID {
		util.WriteMessage(messagesTextView, message.Message)
	}
}

func onGuildsDropDownSelected(text string, _ int) {
	// Remove/clear all items from the channels List
	channelsList.Clear()
	// Remove/clear all text from the messages TextView buffer
	messagesTextView.Clear()
	// If the message InputField is not nil, remove the message InputField from the main Flex and set the message InputField to nil
	if messageInputField != nil {
		mainFlex.RemoveItem(messageInputField)
		messageInputField = nil
	}

	for i := 0; i < len(guilds); i++ {
		guild := guilds[i]
		if guild.Name == text {
			currentGuild = guild
			break
		}
	}

	for i := 0; i < len(currentGuild.Channels); i++ {
		channel := currentGuild.Channels[i]
		channelsList.AddItem(channel.Name, "", 0, nil)
	}

	app.SetFocus(channelsList)
}

func onChannelsListSelected(i int, mainText string, secondaryText string, _ rune) {
	// Remove/clear all text from the messages TextView buffer
	messagesTextView.Clear()
	// If the message InputField is nil, add a new message InputField to the main Flex and assign it to message InputField in instance
	if messageInputField == nil {
		messageInputField = ui.NewMessageInputField(onMessageInputFieldDone)
		// Add the message InputField as a new item to the main Flex
		mainFlex.AddItem(messageInputField, 3, 1, false)
	}

	app.SetFocus(messageInputField)

	currentChannel = currentGuild.Channels[i]
	// Set the title of the messages TextView Box to the name of the channel
	messagesTextView.SetTitle(currentChannel.Name)

	messages := util.GetMessages(discordSession, currentChannel.ID, 50)
	for i := len(messages) - 1; i >= 0; i-- {
		util.WriteMessage(messagesTextView, messages[i])
	}
}

func onMessageInputFieldDone(key tcell.Key) {
	if key == tcell.KeyEnter {
		currentText := messageInputField.GetText()
		currentText = strings.TrimSpace(currentText)
		// If the current text of the message InputField is an empty string and the enter key is pressed, do not proceed
		if currentText == "" {
			return
		}

		util.SendMessage(discordSession, currentChannel.ID, currentText)
		// Set the current text of the message InputField to an empty string after the message has been sent
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

		discordSession = newSession(email, password, "")

		util.SetPassword("email", email)
		util.SetPassword("password", password)
	} else if loginVia == "token" {
		token := loginForm.GetFormItemByLabel("Token").(*tview.InputField).GetText()
		if token == "" {
			return
		}

		discordSession = newSession("", "", token)

		util.SetPassword("token", token)
	}

	app.
		SetRoot(mainFlex, true).
		SetFocus(guildsDropDown)
}
