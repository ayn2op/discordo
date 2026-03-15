package chat

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/flex"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/layers"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/ningen/v3"
	"github.com/diamondburned/ningen/v3/states/read"
	"github.com/gdamore/tcell/v3"
)

const typingDuration = 10 * time.Second

const (
	flexLayerName            = "flex"
	mentionsListLayerName    = "mentionsList"
	attachmentsListLayerName = "attachmentsList"
	confirmModalLayerName    = "confirmModal"
	channelsPickerLayerName  = "channelsPicker"
)

type Model struct {
	*layers.Layers

	// guildsTree (sidebar) + rightFlex
	mainFlex *flex.Model
	// messagesList + messageInput
	rightFlex *flex.Model

	guildsTree     *guildsTree
	messagesList   *messagesList
	messageInput   *messageInput
	channelsPicker *channelsPicker

	selectedChannel   *discord.Channel
	selectedChannelMu sync.RWMutex

	typersMu sync.RWMutex
	typers   map[discord.UserID]*time.Timer

	confirmModalDone          func(label string)
	confirmModalPreviousFocus tview.Primitive

	app   *tview.Application
	cfg   *config.Config
	state *ningen.State
	token string
}

func NewView(app *tview.Application, cfg *config.Config, token string) *Model {
	v := &Model{
		Layers: layers.New(),

		mainFlex:  flex.NewModel(),
		rightFlex: flex.NewModel(),

		typers: make(map[discord.UserID]*time.Timer),

		app:   app,
		cfg:   cfg,
		token: token,
	}

	v.guildsTree = newGuildsTree(cfg, v)
	v.messagesList = newMessagesList(cfg, v)
	v.messageInput = newMessageInput(cfg, v)
	v.channelsPicker = newChannelsPicker(cfg, v)
	v.channelsPicker.SetCancelFunc(v.closePicker)

	v.SetBackgroundLayerStyle(v.cfg.Theme.Dialog.BackgroundStyle.Style)
	v.buildLayout()
	return v
}

func (v *Model) SelectedChannel() *discord.Channel {
	v.selectedChannelMu.RLock()
	defer v.selectedChannelMu.RUnlock()
	return v.selectedChannel
}

func (v *Model) SetSelectedChannel(channel *discord.Channel) {
	v.selectedChannelMu.Lock()
	v.selectedChannel = channel
	v.selectedChannelMu.Unlock()
}

func (v *Model) buildLayout() {
	v.Clear()
	v.rightFlex.Clear()
	v.mainFlex.Clear()

	v.rightFlex.
		SetDirection(flex.DirectionRow).
		AddItem(v.messagesList, 0, 1, false).
		AddItem(v.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	v.mainFlex.
		AddItem(v.guildsTree, 0, 1, true).
		AddItem(v.rightFlex, 0, 4, false)

	v.AddLayer(v.mainFlex, layers.WithName(flexLayerName), layers.WithResize(true), layers.WithVisible(true))
	v.AddLayer(v.messageInput.mentionsList, layers.WithName(mentionsListLayerName), layers.WithResize(false), layers.WithVisible(false))
}

func (v *Model) togglePicker() {
	if v.HasLayer(channelsPickerLayerName) {
		v.closePicker()
	} else {
		v.openPicker()
	}
}

func (v *Model) openPicker() {
	v.AddLayer(
		ui.Centered(v.channelsPicker, v.cfg.Picker.Width, v.cfg.Picker.Height),
		layers.WithName(channelsPickerLayerName),
		layers.WithResize(true),
		layers.WithVisible(true),
		layers.WithOverlay(),
	).SendToFront(channelsPickerLayerName)
	v.channelsPicker.update()
}

func (v *Model) closePicker() {
	v.RemoveLayer(channelsPickerLayerName)
	v.channelsPicker.Update()
}

func (v *Model) toggleGuildsTree() {
	// The guilds tree is visible if the number of items is two.
	if v.mainFlex.GetItemCount() == 2 {
		v.mainFlex.RemoveItem(v.guildsTree)
		if v.guildsTree.HasFocus() {
			v.app.SetFocus(v.mainFlex)
		}
	} else {
		v.buildLayout()
		v.app.SetFocus(v.guildsTree)
	}
}

func (v *Model) focusGuildsTree() bool {
	// The guilds tree is not hidden if the number of items is two.
	if v.mainFlex.GetItemCount() == 2 {
		v.app.SetFocus(v.guildsTree)
		return true
	}

	return false
}

func (v *Model) focusMessageInput() bool {
	if !v.messageInput.GetDisabled() {
		v.app.SetFocus(v.messageInput)
		return true
	}

	return false
}

func (v *Model) focusPrevious() {
	switch v.app.Focused() {
	case v.messagesList: // Handle both a.messagesList and a.flex as well as other edge cases (if there is).
		if v.focusGuildsTree() {
			return
		}
		fallthrough
	case v.guildsTree:
		if v.focusMessageInput() {
			return
		}
		fallthrough
	case v.messageInput:
		v.app.SetFocus(v.messagesList)
	}
}

func (v *Model) focusNext() {
	switch v.app.Focused() {
	case v.messagesList:
		if v.focusMessageInput() {
			return
		}
		fallthrough
	case v.messageInput: // Handle both a.messageInput and a.flex as well as other edge cases (if there is).
		if v.focusGuildsTree() {
			return
		}
		fallthrough
	case v.guildsTree:
		v.app.SetFocus(v.messagesList)
	}
}

func (v *Model) HandleEvent(event tcell.Event) tview.Command {
	switch event := event.(type) {
	case *tview.InitEvent:
		return func() tcell.Event {
			if err := v.OpenState(v.token); err != nil {
				slog.Error("failed to open chat state", "err", err)
				return tcell.NewEventError(err)
			}
			return nil
		}
	case *QuitEvent:
		return tview.Batch(
			v.closeState(),
			tview.Quit(),
		)
	case *tview.ModalDoneEvent:
		if v.HasLayer(confirmModalLayerName) {
			v.RemoveLayer(confirmModalLayerName)
			if v.confirmModalPreviousFocus != nil {
				v.app.SetFocus(v.confirmModalPreviousFocus)
			}
			onDone := v.confirmModalDone
			v.confirmModalDone = nil
			v.confirmModalPreviousFocus = nil
			if onDone != nil {
				onDone(event.ButtonLabel)
			}
			return nil
		}
	case *tview.KeyEvent:
		switch {
		case keybind.Matches(event, v.cfg.Keybinds.FocusGuildsTree.Keybind):
			v.messageInput.removeMentionsList()
			v.focusGuildsTree()
			return nil
		case keybind.Matches(event, v.cfg.Keybinds.FocusMessagesList.Keybind):
			v.messageInput.removeMentionsList()
			v.app.SetFocus(v.messagesList)
			return nil
		case keybind.Matches(event, v.cfg.Keybinds.FocusMessageInput.Keybind):
			v.focusMessageInput()
			return nil
		case keybind.Matches(event, v.cfg.Keybinds.FocusPrevious.Keybind):
			v.focusPrevious()
			return nil
		case keybind.Matches(event, v.cfg.Keybinds.FocusNext.Keybind):
			v.focusNext()
			return nil
		case keybind.Matches(event, v.cfg.Keybinds.Logout.Keybind):
			return tview.Batch(v.closeState(), v.logout())
		case keybind.Matches(event, v.cfg.Keybinds.ToggleGuildsTree.Keybind):
			v.toggleGuildsTree()
			return nil
		case keybind.Matches(event, v.cfg.Keybinds.ToggleChannelsPicker.Keybind):
			v.togglePicker()
			return nil
		}
	case *closeLayerEvent:
		if v.HasLayer(event.name) {
			v.HideLayer(event.name)
		}
		return nil
	}
	return v.Layers.HandleEvent(event)
}

func (v *Model) showConfirmModal(prompt string, buttons []string, onDone func(label string)) {
	v.confirmModalPreviousFocus = v.app.Focused()
	v.confirmModalDone = onDone

	modal := tview.NewModal().
		SetText(prompt).
		AddButtons(buttons)
	v.
		AddLayer(
			ui.Centered(modal, 0, 0),
			layers.WithName(confirmModalLayerName),
			layers.WithResize(true),
			layers.WithVisible(true),
			layers.WithOverlay(),
		).
		SendToFront(confirmModalLayerName)
}

func (v *Model) onReadUpdate(event *read.UpdateEvent) {
	v.app.QueueUpdateDraw(func() {
		// Use indexed node lookup to avoid walking the whole tree on every read
		// event. This runs frequently while reading/typing across channels.
		if event.GuildID.IsValid() {
			if guildNode := v.guildsTree.findNodeByReference(event.GuildID); guildNode != nil {
				v.guildsTree.setNodeLineStyle(guildNode, v.guildsTree.getGuildNodeStyle(event.GuildID))
			}
		}

		// Channel style is always updated for the target channel regardless of
		// whether it's in a guild or DM.
		if channelNode := v.guildsTree.findNodeByReference(event.ChannelID); channelNode != nil {
			v.guildsTree.setNodeLineStyle(channelNode, v.guildsTree.getChannelNodeStyle(event.ChannelID))
		}
	})
}

func (v *Model) clearTypers() {
	v.typersMu.Lock()
	for _, timer := range v.typers {
		timer.Stop()
	}
	clear(v.typers)
	v.typersMu.Unlock()
	v.updateFooter()
}

func (v *Model) addTyper(userID discord.UserID) {
	v.typersMu.Lock()
	typer, ok := v.typers[userID]
	if ok {
		typer.Reset(typingDuration)
	} else {
		v.typers[userID] = time.AfterFunc(typingDuration, func() {
			v.removeTyper(userID)
		})
	}
	v.typersMu.Unlock()
	v.updateFooter()
}

func (v *Model) removeTyper(userID discord.UserID) {
	v.typersMu.Lock()
	if typer, ok := v.typers[userID]; ok {
		typer.Stop()
		delete(v.typers, userID)
	}
	v.typersMu.Unlock()
	v.updateFooter()
}

func (v *Model) updateFooter() {
	selectedChannel := v.SelectedChannel()
	if selectedChannel == nil {
		return
	}
	guildID := selectedChannel.GuildID

	v.typersMu.RLock()
	defer v.typersMu.RUnlock()

	var footer string
	if len(v.typers) > 0 {
		var names []string
		for userID := range v.typers {
			var name string
			if guildID.IsValid() {
				member, err := v.state.Cabinet.Member(guildID, userID)
				if err != nil {
					slog.Error("failed to get member from state", "err", err, "guild_id", guildID, "user_id", userID)
					continue
				}

				if member.Nick != "" {
					name = member.Nick
				} else {
					name = member.User.DisplayOrUsername()
				}
			} else {
				for _, recipient := range selectedChannel.DMRecipients {
					if recipient.ID == userID {
						name = recipient.DisplayOrUsername()
						break
					}
				}
			}

			if name != "" {
				names = append(names, name)
			}
		}

		switch len(names) {
		case 1:
			footer = names[0] + " is typing..."
		case 2:
			footer = fmt.Sprintf("%s and %s are typing...", names[0], names[1])
		case 3:
			footer = fmt.Sprintf("%s, %s, and %s are typing...", names[0], names[1], names[2])
		default:
			footer = "Several people are typing..."
		}
	}

	go v.app.QueueUpdateDraw(func() { v.messagesList.SetFooter(footer) })
}
