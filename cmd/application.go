package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/zalando/go-keyring"
)

type application struct {
	*tview.Application

	cfg *config.Config

	pages        *tview.Pages
	flex         *tview.Flex
	guildsTree   *guildsTree
	messagesText *messagesText
	messageInput *messageInput
}

func newApp(cfg *config.Config) *application {
	app := tview.NewApplication()
	a := &application{
		Application: app,

		cfg: cfg,

		pages:        tview.NewPages(),
		flex:         tview.NewFlex(),
		guildsTree:   newGuildsTree(app, cfg),
		messagesText: newMessagesText(app, cfg),
		messageInput: newMessageInput(app, cfg),
	}

	a.EnableMouse(cfg.Mouse)
	a.SetInputCapture(a.onInputCapture)
	a.flex.SetInputCapture(a.onFlexInputCapture)
	return a
}

func (app *application) show(token string) error {
	if token == "" {
		loginForm := login.NewForm(app.cfg, app.Application, func(token string) {
			if err := app.show(token); err != nil {
				slog.Error("failed to show app", "err", err)
				return
			}
		})

		app.SetRoot(loginForm, true)
	} else {
		if err := openState(token); err != nil {
			return err
		}

		app.init()
		app.SetRoot(app.pages, true)
	}

	return nil
}

func (app *application) run(token string) error {
	if err := app.show(token); err != nil {
		return err
	}

	return app.Run()
}

func (a *application) clearPages() {
	for _, name := range a.pages.GetPageNames(false) {
		a.pages.RemovePage(name)
	}
}

func (a *application) init() {
	a.clearPages()
	a.flex.Clear()

	right := tview.NewFlex()
	right.SetDirection(tview.FlexRow)
	right.AddItem(a.messagesText, 0, 1, false)
	right.AddItem(a.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	a.flex.AddItem(a.guildsTree, 0, 1, true)
	a.flex.AddItem(right, 0, 4, false)
	a.pages.AddAndSwitchToPage("flex", a.flex, true)
}

func (app *application) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case app.cfg.Keys.Quit:
		if discordState != nil {
			if err := discordState.Close(); err != nil {
				slog.Error("failed to close the session", "err", err)
			}
		}

		app.Stop()
	case "Ctrl+C":
		// https://github.com/rivo/tview/blob/a64fc48d7654432f71922c8b908280cdb525805c/application.go#L153
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	}

	return event
}

func (app *application) onFlexInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case app.cfg.Keys.FocusGuildsTree:
		app.SetFocus(app.guildsTree)
		return nil
	case app.cfg.Keys.FocusMessagesText:
		app.SetFocus(app.messagesText)
		return nil
	case app.cfg.Keys.FocusMessageInput:
		app.SetFocus(app.messageInput)
		return nil
	case app.cfg.Keys.Logout:
		app.Stop()

		if err := keyring.Delete(consts.Name, "token"); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case app.cfg.Keys.ToggleGuildsTree:
		// The guilds tree is visible if the numbers of items is two.
		if app.flex.GetItemCount() == 2 {
			app.flex.RemoveItem(app.guildsTree)

			if app.guildsTree.HasFocus() {
				app.SetFocus(app.flex)
			}
		} else {
			app.init()
			app.SetFocus(app.guildsTree)
		}

		return nil
	}

	return event
}
