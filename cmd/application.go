package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/login"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
	"github.com/zalando/go-keyring"
)

const (
	flexPageName            = "flex"
	mentionsListPageName    = "mentionsList"
	attachmentsListPageName = "attachmentsList"
	confirmModalPageName    = "confirmModal"
)

type application struct {
	cfg *config.Config

	*tview.Application
	pages        *tview.Pages
	flex         *tview.Flex
	guildsTree   *guildsTree
	messagesList *messagesList
	messageInput *messageInput
}

func newApplication(cfg *config.Config) *application {
	app := &application{
		cfg: cfg,

		Application:  tview.NewApplication(),
		pages:        tview.NewPages(),
		flex:         tview.NewFlex(),
		guildsTree:   newGuildsTree(cfg),
		messagesList: newMessagesList(cfg),
		messageInput: newMessageInput(cfg),
	}

	app.pages.SetInputCapture(app.onPagesInputCapture)
	app.
		EnableMouse(cfg.Mouse).
		SetInputCapture(app.onInputCapture)
	return app
}

func (a *application) run(token string) error {
	if token == "" {
		loginForm := login.NewForm(a.Application, a.cfg, func(token string) {
			if err := a.run(token); err != nil {
				slog.Error("failed to run application", "err", err)
			}
		})

		return a.SetRoot(loginForm, true).Run()
	}

	if err := openState(token); err != nil {
		return err
	}

	a.init()
	return a.SetRoot(a.pages, true).Run()
}

func (a *application) quit() {
	if discordState != nil {
		if err := discordState.Close(); err != nil {
			slog.Error("failed to close the session", "err", err)
		}
	}

	a.Stop()
}

func (a *application) init() {
	a.pages.Clear()
	a.flex.Clear()

	right := tview.NewFlex().
		SetDirection(tview.FlexRow).
		AddItem(a.messagesList, 0, 1, false).
		AddItem(a.messageInput, 3, 1, false)

	// The guilds tree is always focused first at start-up.
	a.flex.
		AddItem(a.guildsTree, 0, 1, true).
		AddItem(right, 0, 4, false)

	a.pages.AddAndSwitchToPage(flexPageName, a.flex, true)
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

func (a *application) onPagesInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case a.cfg.Keys.FocusGuildsTree:
		a.messageInput.removeMentionsList()
		_ = a.focusGuildsTree()
		return nil
	case a.cfg.Keys.FocusMessagesList:
		a.messageInput.removeMentionsList()
		a.SetFocus(a.messagesList)
		return nil
	case a.cfg.Keys.FocusMessageInput:
		a.SetFocus(a.messageInput)
		return nil
	case a.cfg.Keys.FocusPrevious:
		a.focusPrevious()
		return nil
	case a.cfg.Keys.FocusNext:
		a.focusNext()
		return nil
	case a.cfg.Keys.Logout:
		a.quit()

		if err := keyring.Delete(consts.Name, "token"); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case a.cfg.Keys.ToggleGuildsTree:
		a.toggleGuildsTree()
		return nil
	}

	return event
}

func (a *application) toggleGuildsTree() {
	// The guilds tree is visible if the number of items is two.
	if a.flex.GetItemCount() == 2 {
		a.flex.RemoveItem(a.guildsTree)
		if a.guildsTree.HasFocus() {
			a.SetFocus(a.flex)
		}
	} else {
		a.init()
		a.SetFocus(a.guildsTree)
	}
}

func (a *application) focusGuildsTree() bool {
	// The guilds tree is not hidden if the number of items is two.
	if a.flex.GetItemCount() == 2 {
		a.SetFocus(a.guildsTree)
		return true
	}

	return false
}

func (a *application) focusPrevious() {
	switch a.GetFocus() {
	case a.guildsTree:
		a.SetFocus(a.messageInput)
	case a.messagesList: // Handle both a.messagesList and a.flex as well as other edge cases (if there is).
		if ok := a.focusGuildsTree(); !ok {
			a.SetFocus(a.messageInput)
		}
	case a.messageInput:
		a.SetFocus(a.messagesList)
	}
}

func (a *application) focusNext() {
	switch a.GetFocus() {
	case a.guildsTree:
		a.SetFocus(a.messagesList)
	case a.messagesList:
		a.SetFocus(a.messageInput)
	case a.messageInput: // Handle both a.messageInput and a.flex as well as other edge cases (if there is).
		if ok := a.focusGuildsTree(); !ok {
			a.SetFocus(a.messagesList)
		}
	}
}

func (a *application) showConfirmModal(prompt string, buttons []string, onDone func(label string)) {
	previousFocus := a.GetFocus()

	modal := tview.NewModal().
		SetText(prompt).
		AddButtons(buttons).
		SetDoneFunc(func(_ int, buttonLabel string) {
			a.pages.RemovePage(confirmModalPageName).SwitchToPage(flexPageName)
			a.SetFocus(previousFocus)

			if onDone != nil {
				onDone(buttonLabel)
			}
		})

	a.pages.
		AddAndSwitchToPage(confirmModalPageName, ui.Centered(modal, 0, 0), true).
		ShowPage(flexPageName)
}
