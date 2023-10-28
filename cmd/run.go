package cmd

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/constants"
	"github.com/ayn2op/discordo/ui"
	"github.com/rivo/tview"
)

var (
	discordState *State

	cfg      *config.Config
	app      = tview.NewApplication()
	mainFlex *MainFlex
)

func Run(token string) error {
	var err error
	cfg, err = config.Load()
	if err != nil {
		return err
	}

	logPath, err := os.UserCacheDir()
	if err != nil {
		return err
	}

	logPath = filepath.Join(logPath, constants.Name)
	err = os.MkdirAll(logPath, os.ModePerm)
	if err != nil {
		return err
	}

	logPath = filepath.Join(logPath, "logs.txt")
	f, err := os.OpenFile(logPath, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if token == "" {
		lf := ui.NewLoginForm(cfg)

		go func() {
			mainFlex = newMainFlex()
			if err := <-lf.Error; err != nil {
				app.Stop()
				log.Fatal(err)
			}

			if err := openState(<-lf.Token); err != nil {
				app.Stop()
				log.Fatal(err)
			}

			app.QueueUpdateDraw(func() {
				app.SetRoot(mainFlex, true)
			})
		}()

		app.SetRoot(lf, true)
	} else {
		mainFlex = newMainFlex()
		if err := openState(token); err != nil {
			app.Stop()
		}

		app.SetRoot(mainFlex, true)
	}

	app.EnableMouse(cfg.Mouse)
	return app.Run()
}
