package cmd

import (
	"log/slog"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
)

const (
	flexPageName            = "flex"
	mentionsListPageName    = "mentionsList"
	attachmentsListPageName = "attachmentsList"
	confirmModalPageName    = "confirmModal"
)

type chatView struct {
	*tview.Pages

	mainFlex  *tview.Flex
	rightFlex *tview.Flex

	guildsTree   *guildsTree
	messagesList *messagesList
	messageInput *messageInput

	selectedGuildID   discord.GuildID
	selectedChannelID discord.ChannelID

	app *tview.Application
	cfg *config.Config
}

func newChatView(app *tview.Application, cfg *config.Config) *chatView {
	cv := &chatView{
		Pages: tview.NewPages(),

		mainFlex:  tview.NewFlex(),
		rightFlex: tview.NewFlex(),

		guildsTree:   newGuildsTree(cfg),
		messagesList: newMessagesList(cfg),
		messageInput: newMessageInput(cfg),

		app: app,
		cfg: cfg,
	}

	cv.SetInputCapture(cv.onInputCapture)

	cv.buildLayout()
	return cv
}

func (cv *chatView) buildLayout() {
	cv.Clear()
	cv.rightFlex.Clear()
	cv.mainFlex.Clear()

	cv.rightFlex.
		SetDirection(tview.FlexRow).
		AddItem(cv.messagesList, 0, 1, false).
		AddItem(cv.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	cv.mainFlex.
		AddItem(cv.guildsTree, 0, 1, true).
		AddItem(cv.rightFlex, 0, 4, false)

	cv.AddAndSwitchToPage(flexPageName, cv.mainFlex, true)
}

func (cv *chatView) toggleGuildsTree() {
	// The guilds tree is visible if the number of items is two.
	if cv.mainFlex.GetItemCount() == 2 {
		cv.mainFlex.RemoveItem(cv.guildsTree)
		if cv.guildsTree.HasFocus() {
			cv.app.SetFocus(cv.mainFlex)
		}
	} else {
		cv.buildLayout()
		cv.app.SetFocus(cv.guildsTree)
	}
}

func (cv *chatView) focusGuildsTree() bool {
	// The guilds tree is not hidden if the number of items is two.
	if cv.mainFlex.GetItemCount() == 2 {
		cv.app.SetFocus(cv.guildsTree)
		return true
	}

	return false
}

func (cv *chatView) focusMessageInput() bool {
	if !cv.messageInput.GetDisabled() {
		cv.app.SetFocus(cv.messageInput)
		return true
	}

	return false
}

func (cv *chatView) focusPrevious() {
	switch cv.app.GetFocus() {
	case cv.guildsTree:
		cv.app.SetFocus(cv.messageInput)
	case cv.messagesList: // Handle both a.messagesList and a.flex as well as other edge cases (if there is).
		if ok := cv.focusGuildsTree(); !ok {
			cv.app.SetFocus(cv.messageInput)
		}
	case cv.messageInput:
		cv.app.SetFocus(cv.messagesList)
	}
}

func (cv *chatView) focusNext() {
	switch cv.app.GetFocus() {
	case cv.guildsTree:
		cv.app.SetFocus(cv.messagesList)
	case cv.messagesList:
		cv.app.SetFocus(cv.messageInput)
	case cv.messageInput: // Handle both a.messageInput and a.flex as well as other edge cases (if there is).
		if ok := cv.focusGuildsTree(); !ok {
			cv.app.SetFocus(cv.messagesList)
		}
	}
}

func (cv *chatView) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cv.cfg.Keys.FocusGuildsTree:
		cv.messageInput.removeMentionsList()
		cv.focusGuildsTree()
		return nil
	case cv.cfg.Keys.FocusMessagesList:
		cv.messageInput.removeMentionsList()
		cv.app.SetFocus(cv.messagesList)
		return nil
	case cv.cfg.Keys.FocusMessageInput:
		cv.focusMessageInput()
		return nil
	case cv.cfg.Keys.FocusPrevious:
		cv.focusPrevious()
		return nil
	case cv.cfg.Keys.FocusNext:
		cv.focusNext()
		return nil
	case cv.cfg.Keys.Logout:
		app.quit()

		if err := keyring.DeleteToken(); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case cv.cfg.Keys.ToggleGuildsTree:
		cv.toggleGuildsTree()
		return nil
	}

	return event
}

func (cv *chatView) showConfirmModal(prompt string, buttons []string, onDone func(label string)) {
	previousFocus := cv.app.GetFocus()

	modal := tview.NewModal().
		SetText(prompt).
		AddButtons(buttons).
		SetDoneFunc(func(_ int, buttonLabel string) {
			cv.RemovePage(confirmModalPageName).SwitchToPage(flexPageName)
			cv.app.SetFocus(previousFocus)

			if onDone != nil {
				onDone(buttonLabel)
			}
		})

	cv.
		AddAndSwitchToPage(confirmModalPageName, ui.Centered(modal, 0, 0), true).
		ShowPage(flexPageName)
}
