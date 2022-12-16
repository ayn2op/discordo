package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/zalando/go-keyring"
)

var tokenFlag string

func init() {
	flag.StringVar(&tokenFlag, "token", "", "The authentication token.")

	path, _ := os.UserCacheDir()
	path = filepath.Join(path, config.Name+".log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		log.Fatal(err)
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Llongfile)
}

func main() {
	flag.Parse()

	cfg := config.New()
	if err := cfg.Load(); err != nil {
		log.Fatal(err)
	}

	if tokenFlag != "" {
		go keyring.Set(config.Name, "token", tokenFlag)
	} else {
		var err error
		tokenFlag, err = keyring.Get(config.Name, "token")
		if err != nil {
			log.Println(err)
		}
	}

	app := ui.NewApplication(cfg)
	app.Run(tokenFlag)
}
