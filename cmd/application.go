package cmd

import (
	"fmt"
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
}

func newApplication(cfg *config.Config) *application {
	app := &application{
		Application: tview.NewApplication(),

		cfg: cfg,

		pages:        tview.NewPages(),
		flex:         tview.NewFlex(),
		guildsTree:   newGuildsTree(cfg),
		messagesText: newMessagesText(cfg),
		messageInput: newMessageInput(cfg),
	}

	app.EnableMouse(cfg.Mouse)
	app.SetInputCapture(app.onInputCapture)
	app.flex.SetInputCapture(app.onFlexInputCapture)
	return app
}

func (a *application) show(token string) error {
	if token == "" {
		loginForm := login.NewForm(a.cfg, a.Application, func(token string) {
			if err := a.show(token); err != nil {
				slog.Error("failed to show app", "err", err)
				return
			}
		})

		a.SetRoot(loginForm, true)
	} else {
		if err := openState(token); err != nil {
			return err
		}

		a.init()
		a.SetRoot(a.pages, true)
	}

	return nil
}

func (a *application) run(token string) error {
	if err := a.show(token); err != nil {
		return err
	}

	if err := a.Run(); err != nil {
		return fmt.Errorf("failed to run application: %w", err)
	}

	return nil
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
	a.pages.AddPage("candidates", a.messageInput.candidates, false, false)
}

func (a *application) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case a.cfg.Keys.Quit:
		if discordState != nil {
			if err := discordState.Close(); err != nil {
				slog.Error("failed to close the session", "err", err)
			}
		}

		a.Stop()
	case "Ctrl+C":
		// https://github.com/ayn2op/tview/blob/a64fc48d7654432f71922c8b908280cdb525805c/application.go#L153
		return tcell.NewEventKey(tcell.KeyCtrlC, 0, tcell.ModNone)
	}

	return event
}

func (a *application) onFlexInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case a.cfg.Keys.FocusGuildsTree:
		app.pages.HidePage("candidates")
		app.SetFocus(app.guildsTree)
		return nil
	case a.cfg.Keys.FocusMessagesText:
		app.pages.HidePage("candidates")
		app.SetFocus(app.messagesText)
		return nil
	case a.cfg.Keys.FocusMessageInput:
		a.SetFocus(a.messageInput)
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
			a.flex.RemoveItem(a.guildsTree)

			if a.guildsTree.HasFocus() {
				a.SetFocus(a.flex)
			}
		} else {
			a.init()
			a.SetFocus(a.guildsTree)
		}

		return nil
	}

	return event
}
