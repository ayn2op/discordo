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
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/ningen/v3"
	"github.com/diamondburned/ningen/v3/states/read"
	"github.com/gdamore/tcell/v3"
)

const typingDuration = 10 * time.Second

const (
	flexPageName            = "flex"
	mentionsListPageName    = "mentionsList"
	attachmentsListPageName = "attachmentsList"
	confirmModalPageName    = "confirmModal"
)

type View struct {
	*tview.Pages

	mainFlex  *tview.Flex
	rightFlex *tview.Flex

	guildsTree   *guildsTree
	messagesList *messagesList
	messageInput *messageInput

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
		Pages: tview.NewPages(),

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

	v.SetInputCapture(v.onInputCapture)
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

	v.AddAndSwitchToPage(flexPageName, v.mainFlex, true)
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
	case v.guildsTree:
		v.focusMessageInput()
	case v.messagesList: // Handle both a.messagesList and a.flex as well as other edge cases (if there is).
		if ok := v.focusGuildsTree(); !ok {
			v.app.SetFocus(v.messageInput)
		}
	case v.messageInput:
		v.app.SetFocus(v.messagesList)
	}
}

func (v *View) focusNext() {
	switch v.app.GetFocus() {
	case v.guildsTree:
		v.app.SetFocus(v.messagesList)
	case v.messagesList:
		v.focusMessageInput()
	case v.messageInput: // Handle both a.messageInput and a.flex as well as other edge cases (if there is).
		if ok := v.focusGuildsTree(); !ok {
			v.app.SetFocus(v.messagesList)
		}
	}
}

func (v *View) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case v.cfg.Keys.FocusGuildsTree:
		v.messageInput.removeMentionsList()
		v.focusGuildsTree()
		return nil
	case v.cfg.Keys.FocusMessagesList:
		v.messageInput.removeMentionsList()
		v.app.SetFocus(v.messagesList)
		return nil
	case v.cfg.Keys.FocusMessageInput:
		v.focusMessageInput()
		return nil
	case v.cfg.Keys.FocusPrevious:
		v.focusPrevious()
		return nil
	case v.cfg.Keys.FocusNext:
		v.focusNext()
		return nil
	case v.cfg.Keys.Logout:
		if v.onLogout != nil {
			v.onLogout()
		}

		if err := keyring.DeleteToken(); err != nil {
			slog.Error("failed to delete token from keyring", "err", err)
			return nil
		}

		return nil
	case v.cfg.Keys.ToggleGuildsTree:
		v.toggleGuildsTree()
		return nil
	}

	return event
}

func (v *View) showConfirmModal(prompt string, buttons []string, onDone func(label string)) {
	previousFocus := v.app.GetFocus()

	modal := tview.NewModal().
		SetText(prompt).
		AddButtons(buttons).
		SetDoneFunc(func(_ int, buttonLabel string) {
			v.RemovePage(confirmModalPageName).SwitchToPage(flexPageName)
			v.app.SetFocus(previousFocus)

			if onDone != nil {
				onDone(buttonLabel)
			}
		})

	v.
		AddAndSwitchToPage(confirmModalPageName, ui.Centered(modal, 0, 0), true).
		ShowPage(flexPageName)
}

func (v *View) onReadUpdate(event *read.UpdateEvent) {
	channel, err := v.state.Cabinet.Channel(event.ChannelID)
	if err != nil {
		return
	}
	
	isDM := channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM
	
	v.app.QueueUpdateDraw(func() {
		var guildNode *tview.TreeNode
		var dmNode *tview.TreeNode
		
		v.guildsTree.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {
			if node.GetReference() == event.GuildID {
				guildNode = node
				return false
			}
			if node.GetReference() == nil && node.GetText() == "Direct Messages" {
				dmNode = node
			}
			return true
		})

		if isDM {
			if dmNode != nil {
				var found bool
				dmNode.Walk(func(node, parent *tview.TreeNode) bool {
					if node.GetReference() == event.ChannelID {
						node.SetTextStyle(v.guildsTree.getChannelNodeStyle(event.ChannelID))
						found = true
						return false
					}
					if chID, ok := node.GetReference().(discord.ChannelID); ok && chID == event.ChannelID {
						node.SetTextStyle(v.guildsTree.getChannelNodeStyle(event.ChannelID))
						found = true
						return false
					}
					return true
				})
				
				if !found {
					v.guildsTree.createChannelNode(dmNode, *channel)
				}
			}
		} else if guildNode != nil {
			guildNode.SetTextStyle(v.guildsTree.getGuildNodeStyle(event.GuildID))
			guildNode.Walk(func(node, parent *tview.TreeNode) bool {
				if node.GetReference() == event.ChannelID {
					node.SetTextStyle(v.guildsTree.getChannelNodeStyle(event.ChannelID))
					return false
				}
				return true
			})
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

func (v *View) refreshChannelNodeStyle(channelID discord.ChannelID) {
	channel, err := v.state.Cabinet.Channel(channelID)
	if err != nil {
		return
	}
	
	isDM := channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM
	
	var dmNode *tview.TreeNode
	var guildNode *tview.TreeNode
	
	v.guildsTree.GetRoot().Walk(func(node, parent *tview.TreeNode) bool {
		if node.GetReference() == nil && node.GetText() == "Direct Messages" {
			dmNode = node
		}
		if guildID, ok := node.GetReference().(discord.GuildID); ok {
			if channel.GuildID.IsValid() && guildID == channel.GuildID {
				guildNode = node
			}
		}
		return true
	})
	
	if isDM {
		if dmNode == nil {
			return
		}
		var found bool
		dmNode.Walk(func(node, parent *tview.TreeNode) bool {
			if node.GetReference() == channelID {
				readState := v.state.ReadState.ReadState(channelID)
				var indication ningen.UnreadIndication
				if readState == nil || !readState.LastMessageID.IsValid() {
					indication = ningen.ChannelRead
				} else if readState.MentionCount > 0 {
					indication = ningen.ChannelMentioned
				} else if readState.LastMessageID < channel.LastMessageID {
					indication = ningen.ChannelUnread
				} else {
					indication = ningen.ChannelRead
				}
				node.SetTextStyle(v.guildsTree.unreadStyle(indication))
				found = true
				return false
			}
			return true
		})
		if !found {
			v.guildsTree.createChannelNode(dmNode, *channel)
		}
	} else if guildNode != nil {
		guildNode.Walk(func(childNode, _ *tview.TreeNode) bool {
			if childNode.GetReference() == channelID {
				readState := v.state.ReadState.ReadState(channelID)
				var indication ningen.UnreadIndication
				if readState != nil && readState.LastMessageID.IsValid() {
					if readState.MentionCount > 0 {
						indication = ningen.ChannelMentioned
					} else if readState.LastMessageID < channel.LastMessageID {
						indication = ningen.ChannelUnread
					} else {
						indication = ningen.ChannelRead
					}
				} else {
					indication = v.state.ChannelIsUnread(channelID, ningen.UnreadOpts{IncludeMutedCategories: true})
				}
				childNode.SetTextStyle(v.guildsTree.unreadStyle(indication))
				return false
			}
			return true
		})
	}
}
