package main

import (
	"log"
	"os"

	"github.com/alecthomas/kong"
	"github.com/ayntgl/discordo/config"
	"github.com/ayntgl/discordo/ui"
	"github.com/zalando/go-keyring"
)

var cli struct {
	Token      string `name:"token" short:"t" help:"The authentication token."`
	ConfigPath string `name:"config" short:"c" help:"The path to the configuration file" type:"path" default:"${configPath}"`
	LogPath    string `name:"log" short:"l" help:"The path to the log file" type:"path" default:"${logPath}"`
}

func main() {
	kong.Parse(&cli, kong.Vars{
		"configPath": config.DefaultConfigPath(),
		"logPath":    config.DefaultLogPath(),
	})

	if cli.LogPath != "" {
		// Set the standard logger output to the provided log file.
		f, err := os.OpenFile(cli.LogPath, os.O_CREATE|os.O_WRONLY, 0666)
		if err != nil {
			log.Fatal(err)
		}

		log.SetOutput(f)
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}

	cfg := config.New()
	if err := cfg.Load(cli.ConfigPath); err != nil {
		log.Fatal(err)
	}

	if cli.Token != "" {
		go keyring.Set(config.Name, "token", cli.Token)
	} else {
		var err error
		cli.Token, err = keyring.Get(config.Name, "token")
		if err != nil {
			log.Println(err)
		}
	}

	app := ui.NewApplication(cfg)
	app.Run(cli.Token)
}
