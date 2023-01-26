package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

var (
	token string

	discordState *State

	app      = tview.NewApplication()
	mainFlex *MainFlex
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
	if token != "" {
		go keyring.Set(config.Name, "token", token)
	} else {
		token, err = keyring.Get(config.Name, "token")
		if err != nil {
			log.Println(err)
		}
	}

	if err = config.Load(); err != nil {
		log.Fatal(err)
	}

	// mission failed, we'll get 'em next time
	if token == "" {
		app.SetRoot(newLoginForm(), true)
	} else {
		mainFlex = newMainFlex()

		discordState, err = openState(token)
		if err != nil {
			log.Fatal(err)
		}

		app.SetRoot(mainFlex, true)
	}

	app.EnableMouse(config.Current.Mouse)
	err = app.Run()
	if err != nil {
		log.Fatal(err)
	}
}
