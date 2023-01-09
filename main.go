package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/config"
	"github.com/zalando/go-keyring"
)

var tokenFlag string

func init() {
	flag.StringVar(&tokenFlag, "token", "", "The authentication token.")

	path := config.LogDirPath()
	err := os.MkdirAll(path, os.ModePerm)
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
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
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Llongfile)
}

func main() {
	flag.Parse()

	var err error
	// If the token is passed via the flag, set it in the keyring.
	if token != "" {
		go keyring.Set(name, "token", token)
	} else {
		token, err = keyring.Get(name, "token")
		if err != nil {
			log.Println(err)
			return
		}
	}

	config, err = newConfig()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize UI
	guildsTree = newGuildsTree()
	messagesText = newMessagesText()
	messageInput = newMessageInput()

	// mission failed, we'll get 'em next time
	if token == "" {
		app.SetRoot(newLoginForm(), true)
	} else {
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

		app.SetRoot(flex, true)
	}

	app.EnableMouse(config.Mouse)
	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
