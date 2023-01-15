package main

import (
	"context"
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/config"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

var (
	token string

	cfg          *config.Config
	discordState *State

	app          = tview.NewApplication()
	mainFlex     = tview.NewFlex()
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

	path = filepath.Join(path, config.Name)
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
		go keyring.Set(config.Name, "token", token)
	} else {
		token, err = keyring.Get(config.Name, "token")
		if err != nil {
			log.Println(err)
			return
		}
	}

	cfg, err = config.Load()
	if err != nil {
		log.Fatal(err)
	}

	// Initialize UI widgets
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
		mainFlex.AddItem(guildsTree, 0, 1, true)
		mainFlex.AddItem(right, 0, 4, false)

		mainFlex.SetInputCapture(onInputCapture)
		app.SetRoot(mainFlex, true)
	}

	app.EnableMouse(cfg.Mouse)
	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
