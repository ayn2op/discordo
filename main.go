package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/rivo/tview"
)

var (
	token string

	config       *Config
	discordState *State

	app  = tview.NewApplication()
	flex = tview.NewFlex()

	guildsTree   *GuildsTree
	messagesText *MessagesText
	messageInput *MessageInput
)

func init() {
	flag.StringVar(&token, "token", "", "The authentication token.")

	path, err := os.UserCacheDir()
	if err != nil {
		log.Fatal(err)
	}

	path = filepath.Join(path, name)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	path = filepath.Join(path, "logs.txt")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Llongfile)
}

func main() {
	flag.Parse()

	var err error
	config, err = newConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize UI
	guildsTree = newGuildsTree()
	messagesText = newMessagesText()
	messageInput = newMessageInput()

	discordState = newState(token)
	err = discordState.Open(context.Background())
	if err != nil {
		log.Fatal(err)
	}

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(messagesText, 0, 1, false)
	right.AddItem(messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	flex.AddItem(guildsTree, 0, 1, true)
	flex.AddItem(right, 0, 4, false)

	app.EnableMouse(config.Mouse)
	app.SetRoot(flex, true)

	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
