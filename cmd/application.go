package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
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

	flexPage         *tview.Page
	autocompletePage *tview.Page
}

func newApp(cfg *config.Config) *application {
	app := tview.NewApplication()
	a := &application{
		Application: app,

		cfg: cfg,

		pages:        tview.NewPages(),
		flex:         tview.NewFlex(),
		guildsTree:   newGuildsTree(cfg),
		messagesText: newMessagesText(cfg),
		messageInput: newMessageInput(cfg),
	}

	a.EnableMouse(cfg.Mouse)
	a.SetInputCapture(a.onInputCapture)
	a.flex.SetInputCapture(a.onFlexInputCapture)
	return a
}

func (app *application) show(token string) error {
	if token == "" {
		loginForm := login.NewForm(app.cfg, func(token string) {
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
	for _, p := range a.pages.GetPages() {
		a.pages.RemovePage(p)
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
	a.flexPage = a.pages.AddAndSwitchToPage(a.flex, true)
	a.autocompletePage = a.pages.AddPage(a.messageInput.autocomplete, false, false)
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
		// https://github.com/ayn2op/tview/blob/a64fc48d7654432f71922c8b908280cdb525805c/application.go#L153
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	}

	return event
}

func (a *application) onFlexInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case a.cfg.Keys.FocusGuildsTree:
		a.pages.HidePage(a.autocompletePage)
		a.SetFocus(app.guildsTree)
		return nil
	case a.cfg.Keys.FocusMessagesText:
		a.pages.HidePage(a.autocompletePage)
		a.SetFocus(app.messagesText)
		return nil
	case a.cfg.Keys.FocusMessageInput:
		a.SetFocus(app.messageInput)
		return nil
	case a.cfg.Keys.Logout:
		a.Stop()

		if err := keyring.Delete(consts.Name, "token"); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case a.cfg.Keys.ToggleGuildsTree:
		// The guilds tree is visible if the numbers of items is two.
		if a.flex.GetItemCount() == 2 {
			a.flex.RemoveItem(app.guildsTree)

			if a.guildsTree.HasFocus() {
				a.SetFocus(app.flex)
			}
		} else {
			a.init()
			a.SetFocus(a.guildsTree)
		}

		return nil
	}

	return event
}
