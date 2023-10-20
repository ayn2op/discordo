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
}

func (r *Cmd) Run() error {
	path := config.DefaultPath()
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}

	err = config.Load(path)
	if err != nil {
		return err
	}

	path = config.DefaultLogPath()
	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}

	f, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY, os.ModePerm)
	if err != nil {
		return err
	}

	log.SetOutput(f)
	log.SetFlags(log.LstdFlags | log.Llongfile)

	if r.Token == "" {
		lf := ui.NewLoginForm()

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
		if err := openState(r.Token); err != nil {
			app.Stop()
		}

		app.SetRoot(mainFlex, true)
	}

	app.EnableMouse(config.Current.Mouse)
	return app.Run()
}
