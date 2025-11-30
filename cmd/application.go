package cmd

import (
	"fmt"
	"log/slog"
	"reflect"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
	"golang.design/x/clipboard"
)

type application struct {
	*tview.Application
	chatView *chatView
	cfg      *config.Config
}

func newApplication(cfg *config.Config) *application {
	tview.Styles = tview.Theme{}
	app := &application{
		Application: tview.NewApplication(),
		cfg:         cfg,
	}

	if err := clipboard.Init(); err != nil {
		slog.Error("failed to init clipboard", "err", err)
	}

	app.
		EnableMouse(cfg.Mouse).
		SetInputCapture(app.onInputCapture).
		SetBeforeDrawFunc(app.onBeforeDraw).
		EnablePaste(true)
	return app
}

func (a *application) run(token string) error {
	if token == "" {
		loginForm := login.NewForm(a.Application, a.cfg, func(token string) {
			if err := a.run(token); err != nil {
				slog.Error("failed to run application", "err", err)
			}
		})
		a.SetRoot(loginForm, true)
	} else {
		a.chatView = newChatView(a.Application, a.cfg)
		if err := openState(token); err != nil {
			return err
		}

		a.SetRoot(a.chatView, true)
	}

	return a.Run()
}

func (a *application) quit() {
	if discordState != nil {
		if err := discordState.Close(); err != nil {
			slog.Error("failed to close the session", "err", err)
		}
	}

	a.Stop()
}

func (a *application) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case a.cfg.Keys.Quit:
		a.quit()
		return nil
	case "Ctrl+C":
		// https://github.com/ayn2op/tview/blob/a64fc48d7654432f71922c8b908280cdb525805c/application.go#L153
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	}

	return event
}

func (a *application) onBeforeDraw(screen tcell.Screen) bool {
	if a.chatView != nil {
		var f = a.GetFocus()
		switch f {
			// ideally these are NOT hardcoded rofl
			case a.chatView.guildsTree:
				a.chatView.statusBar.setText("k prev j next g first G last RTN select")
			case a.chatView.messagesList:
				a.chatView.statusBar.setText("k prev j next g first G last r reply R @reply")
			case a.chatView.messageInput:
				a.chatView.statusBar.setText("RTN send ALT-RTN newline ESC clear CTRL-\\ attach")
			default:
				// mouse input seems to cause this case, not sure of a solution :(
				a.chatView.statusBar.setText(fmt.Sprint(reflect.TypeOf(f)))
		}
	}
	return false
}
