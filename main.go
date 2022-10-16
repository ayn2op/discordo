package main

import (
	"flag"
	"log"
	"os"

	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/zalando/go-keyring"
)

var (
	flagToken  string
	flagConfig string
	flagLog    string
)

func init() {
	flag.StringVar(&flagToken, "token", "", "The authentication token.")
	flag.StringVar(&flagConfig, "config", config.DefaultConfigPath(), "The path to the configuration file.")
	flag.StringVar(&flagLog, "log", config.DefaultLogPath(), "The path to the log file.")
}

func main() {
	flag.Parse()

	if flagLog != "" {
		// Set the standard logger output to the provided log file.
		f, err := os.OpenFile(flagLog, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}

		log.SetOutput(f)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	cfg := config.New()
	if err := cfg.Load(flagConfig); err != nil {
		log.Fatal(err)
	}

	var (
		token string
		err   error
	)

	if flagToken != "" {
		token = flagToken
		go keyring.Set(config.Name, "token", token)
	} else {
		token, err = keyring.Get(config.Name, "token")
		if err != nil {
			log.Println(err)
		}
	}

	app := ui.NewApplication(cfg)
	app.Run(token)
}
