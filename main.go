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

	app          *tview.Application
	flex         *tview.Flex
	guildsTree   *GuildsTree
	channelsTree *ChannelsTree
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

	// Initialize UI
	app = tview.NewApplication()

	guildsTree = newGuildsTree()
	channelsTree = newChannelsTree()
	messagesText = newMessagesText()
	messageInput = newMessageInput()

	discordState = newState(token)
	if err = discordState.Open(context.Background()); err != nil {
		log.Fatal(err)
	}

	left := tview.NewFlex()
	left.SetDirection(tview.FlexRow)
	left.AddItem(guildsTree, 0, 1, true)
	left.AddItem(channelsTree, 0, 1, false)

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(messagesText, 0, 1, false)
	right.AddItem(messageInput, 3, 1, false)

	flex = tview.NewFlex()
	flex.AddItem(left, 0, 1, false)
	flex.AddItem(right, 0, 4, false)

	app.EnableMouse(cfg.Mouse)
	app.SetRoot(flex, true)
	if err = app.Run(); err != nil {
		log.Fatal(err)
	}
}
