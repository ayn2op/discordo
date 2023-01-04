package main

import (
	"context"
	"flag"
	"log"

	"github.com/rivo/tview"
)

var (
	token string

	cfg          *Config
	discordState *State

	app  = tview.NewApplication()
	flex = tview.NewFlex()

	guildsTree   *GuildsTree
	messagesText *MessagesText
	messageInput *MessageInput
)

func init() {
	flag.StringVar(&token, "token", "", "The authentication token.")
}

func main() {
	flag.Parse()

	var err error
	cfg, err = newConfig()
	if err != nil {
		log.Fatal(err)
	}

	discordState = newState(token)
	if err = discordState.Open(context.Background()); err != nil {
		log.Fatal(err)
	}

	// Initialize UI
	guildsTree = newGuildsTree()
	messagesText = newMessagesText()
	messageInput = newMessageInput()

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(messagesText, 0, 1, false)
	right.AddItem(messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	flex.AddItem(guildsTree, 0, 1, true)
	flex.AddItem(right, 0, 4, false)

	app.EnableMouse(cfg.Mouse)
	app.SetRoot(flex, true)
	if err = app.Run(); err != nil {
		log.Fatal(err)
	}
}
