package run

import (
	"log"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/config"
	"github.com/ayn2op/discordo/ui"
	"github.com/rivo/tview"
)

var (
	discordState *State

	app      = tview.NewApplication()
	mainFlex *MainFlex
)

type Cmd struct {
	Token string `default:"${token}" help:"The authentication token." short:"t"`

	Config string `default:"${configPath}" help:"The path of the configuration file." short:"l" type:"path"`
	Log    string `default:"${logPath}" help:"The path of the log file." short:"c" type:"path"`
}

func (r *Cmd) Run() error {
	err := os.MkdirAll(filepath.Dir(r.Config), os.ModePerm)
	if err != nil {
		return err
	}

	err = config.Load(r.Config)
	if err != nil {
		return err
	}

	err = os.MkdirAll(filepath.Dir(r.Log), os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(r.Log, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if r.Token == "" {
		lf := ui.NewLoginForm()

		go func() {
			mainFlex = newMainFlex()
			if err := openState(<-lf.Token); err != nil {
				log.Fatal(err)
			}

			app.QueueUpdateDraw(func() {
				app.SetRoot(mainFlex, true)
			})
		}()

		app.SetRoot(lf, true)
	} else {
		mainFlex = newMainFlex()
		if err := openState(r.Token); err != nil {
			log.Fatal(err)
		}

		app.SetRoot(mainFlex, true)
	}

	app.EnableMouse(config.Current.Mouse)
	return app.Run()
}
