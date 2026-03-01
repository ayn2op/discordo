package chat

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/keyring"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
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

type View struct {
	*layers.Layers

	rootFlex  *tview.Flex
	mainFlex  *tview.Flex
	rightFlex *tview.Flex

	guildsTree     *guildsTree
	messagesList   *messagesList
	messageInput   *messageInput
	channelsPicker *channelsPicker
	help           *help.Help

	selectedChannel   *discord.Channel
	selectedChannelMu sync.RWMutex

	typersMu sync.RWMutex
	typers   map[discord.UserID]*time.Timer

	app   *tview.Application
	cfg   *config.Config
	state *ningen.State

	onLogout func()
}

func NewView(app *tview.Application, cfg *config.Config, onLogout func()) *View {
	v := &View{
		Layers: layers.New(),

		rootFlex:  tview.NewFlex(),
		mainFlex:  tview.NewFlex(),
		rightFlex: tview.NewFlex(),

		typers: make(map[discord.UserID]*time.Timer),

		app:      app,
		cfg:      cfg,
		onLogout: onLogout,
	}

	v.guildsTree = newGuildsTree(cfg, v)
	v.messagesList = newMessagesList(cfg, v)
	v.messageInput = newMessageInput(cfg, v)
	v.channelsPicker = newChannelsPicker(cfg, v)
	v.channelsPicker.SetCancelFunc(v.closePicker)

	v.help = help.New()

	styles := help.DefaultStyles()
	styles.ShortKeyStyle = cfg.Theme.Help.ShortKeyStyle.Style
	styles.ShortDescStyle = cfg.Theme.Help.ShortDescStyle.Style
	styles.FullKeyStyle = cfg.Theme.Help.FullKeyStyle.Style
	styles.FullDescStyle = cfg.Theme.Help.FullDescStyle.Style
	v.help.SetStyles(styles)

	v.help.SetKeyMap(v)
	v.help.SetCompactModifiers(cfg.Help.CompactModifiers)
	v.help.SetShortSeparator(cfg.Help.Separator)
	v.help.SetBorderPadding(0, 0, cfg.Help.Padding[0], cfg.Help.Padding[1])

	v.SetBackgroundLayerStyle(v.cfg.Theme.Dialog.BackgroundStyle.Style)
	v.buildLayout()
	return v
}

func (v *View) SelectedChannel() *discord.Channel {
	v.selectedChannelMu.RLock()
	defer v.selectedChannelMu.RUnlock()
	return v.selectedChannel
}

func (v *View) SetSelectedChannel(channel *discord.Channel) {
	v.selectedChannelMu.Lock()
	v.selectedChannel = channel
	v.selectedChannelMu.Unlock()
}

func (v *View) buildLayout() {
	v.Clear()
	v.rootFlex.Clear()
	v.rightFlex.Clear()
	v.mainFlex.Clear()

	v.rightFlex.
		SetDirection(tview.FlexRow).
		AddItem(v.messagesList, 0, 1, false).
		AddItem(v.messageInput, 3, 1, false)
	// The guilds tree is always focused first at start-up.
	v.mainFlex.
		AddItem(v.guildsTree, 0, 1, true).
		AddItem(v.rightFlex, 0, 4, false)

	v.rootFlex.
		SetDirection(tview.FlexRow).
		AddItem(v.mainFlex, 0, 1, true).
		AddItem(v.help, 1, 0, false)

	v.updateHelpHeight()
	v.AddLayer(v.rootFlex, layers.WithName(flexLayerName), layers.WithResize(true), layers.WithVisible(true))
	v.AddLayer(v.messageInput.mentionsList, layers.WithName(mentionsListLayerName), layers.WithResize(false), layers.WithVisible(false))
}

func (v *View) togglePicker() {
	if v.HasLayer(channelsPickerLayerName) {
		v.closePicker()
	} else {
		v.openPicker()
	}
}

func (v *View) openPicker() {
	v.AddLayer(
		ui.Centered(v.channelsPicker, v.cfg.Picker.Width, v.cfg.Picker.Height),
		layers.WithName(channelsPickerLayerName),
		layers.WithResize(true),
		layers.WithVisible(true),
		layers.WithOverlay(),
	).SendToFront(channelsPickerLayerName)
	v.channelsPicker.update()
}

func (v *View) closePicker() {
	v.RemoveLayer(channelsPickerLayerName)
	v.channelsPicker.Update()
}

func (v *View) toggleGuildsTree() {
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

func (v *View) focusGuildsTree() bool {
	// The guilds tree is not hidden if the number of items is two.
	if v.mainFlex.GetItemCount() == 2 {
		v.app.SetFocus(v.guildsTree)
		return true
	}

	return false
}

func (v *View) focusMessageInput() bool {
	if !v.messageInput.GetDisabled() {
		v.app.SetFocus(v.messageInput)
		return true
	}

	return false
}

func (v *View) focusPrevious() {
	switch v.app.GetFocus() {
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

func (v *View) focusNext() {
	switch v.app.GetFocus() {
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

func (v *View) InputHandler(event *tcell.EventKey) tview.Command {
	consume := tview.BatchCommand{tview.RedrawCommand{}, tview.ConsumeEventCommand{}}

	switch {
	case keybind.Matches(event, v.cfg.Keybinds.ToggleHelp.Keybind):
		v.help.SetShowAll(!v.help.ShowAll())
		v.updateHelpHeight()
		return consume
	case keybind.Matches(event, v.cfg.Keybinds.FocusGuildsTree.Keybind):
		v.messageInput.removeMentionsList()
		v.focusGuildsTree()
		return consume
	case keybind.Matches(event, v.cfg.Keybinds.FocusMessagesList.Keybind):
		v.messageInput.removeMentionsList()
		v.app.SetFocus(v.messagesList)
		return consume
	case keybind.Matches(event, v.cfg.Keybinds.FocusMessageInput.Keybind):
		v.focusMessageInput()
		return consume
	case keybind.Matches(event, v.cfg.Keybinds.FocusPrevious.Keybind):
		v.focusPrevious()
		return consume
	case keybind.Matches(event, v.cfg.Keybinds.FocusNext.Keybind):
		v.focusNext()
		return consume
	case keybind.Matches(event, v.cfg.Keybinds.Logout.Keybind):
		if v.onLogout != nil {
			v.onLogout()
		}
		if err := keyring.DeleteToken(); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
		}
		return consume
	case keybind.Matches(event, v.cfg.Keybinds.ToggleGuildsTree.Keybind):
		v.toggleGuildsTree()
		return consume
	case keybind.Matches(event, v.cfg.Keybinds.ToggleChannelsPicker.Keybind):
		v.togglePicker()
		return consume
	}

	cmd := v.Layers.InputHandler(event)
	return v.consumeLayerCommands(cmd)
}

func (v *View) consumeLayerCommands(command tview.Command) tview.Command {
	if command == nil {
		return nil
	}

	var commands []tview.Command
	switch c := command.(type) {
	case tview.BatchCommand:
		commands = c
	default:
		commands = []tview.Command{c}
	}

	remaining := make([]tview.Command, 0, len(commands))
	for _, cmd := range commands {
		switch c := cmd.(type) {
		case layers.OpenLayerCommand:
			if v.HasLayer(c.Name) {
				v.ShowLayer(c.Name).SendToFront(c.Name)
			}
			continue
		case layers.CloseLayerCommand:
			if v.HasLayer(c.Name) {
				v.HideLayer(c.Name)
			}
			continue
		case layers.ToggleLayerCommand:
			if v.HasLayer(c.Name) {
				if v.GetVisible(c.Name) {
					v.HideLayer(c.Name)
				} else {
					v.ShowLayer(c.Name).SendToFront(c.Name)
				}
			}
			continue
		}
		remaining = append(remaining, cmd)
	}

	if len(remaining) == 0 {
		return nil
	}
	if len(remaining) == 1 {
		return remaining[0]
	}
	return tview.BatchCommand(remaining)
}

func (v *View) updateHelpHeight() {
	height := 1
	if v.help.ShowAll() {
		height = len(v.help.FullHelpLines(v.FullHelp(), 0))
		if height < 1 {
			height = 1
		}
	}
	v.rootFlex.ResizeItem(v.help, height, 0)
}

func (v *View) showConfirmModal(prompt string, buttons []string, onDone func(label string)) {
	previousFocus := v.app.GetFocus()

	modal := tview.NewModal().
		SetText(prompt).
		AddButtons(buttons).
		SetDoneFunc(func(_ int, buttonLabel string) {
			v.RemoveLayer(confirmModalLayerName)
			v.app.SetFocus(previousFocus)

			if onDone != nil {
				onDone(buttonLabel)
			}
		})
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

func (v *View) onReadUpdate(event *read.UpdateEvent) {
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

func (v *View) clearTypers() {
	v.typersMu.Lock()
	for _, timer := range v.typers {
		timer.Stop()
	}
	clear(v.typers)
	v.typersMu.Unlock()
	v.updateFooter()
}

func (v *View) addTyper(userID discord.UserID) {
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

func (v *View) removeTyper(userID discord.UserID) {
	v.typersMu.Lock()
	if typer, ok := v.typers[userID]; ok {
		typer.Stop()
		delete(v.typers, userID)
	}
	v.typersMu.Unlock()
	v.updateFooter()
}

func (v *View) updateFooter() {
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
			footer = fmt.Sprintf("%s is typing...", names[0])
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
