package ui

import (
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/util"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func onAppInputCapture(app *App, e *tcell.EventKey) *tcell.EventKey {
	if util.HasKeybinding(app.Config.Keybindings.FocusGuildsList, e.Name()) {
		app.SetFocus(app.GuildsList)
		return nil
	} else if util.HasKeybinding(app.Config.Keybindings.FocusChannelsTreeView, e.Name()) {
		app.SetFocus(app.ChannelsTreeView)
		return nil
	} else if util.HasKeybinding(app.Config.Keybindings.FocusMessagesTextView, e.Name()) {
		app.SetFocus(app.MessagesTextView)
		return nil
	} else if util.HasKeybinding(app.Config.Keybindings.FocusMessageInputField, e.Name()) {
		app.SetFocus(app.MessageInputField)
		return nil
	}

	return e
}

func onGuildsListSelected(app *App, guildIdx int) {
	rootTreeNode := app.ChannelsTreeView.GetRoot()
	rootTreeNode.ClearChildren()
	app.MessagesTextView.
		Highlight().
		Clear().
		SetTitle("")
	app.MessageInputField.SetText("")

	if guildIdx == 0 { // Direct Messages
		cs := app.Session.State.PrivateChannels
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].LastMessageID > cs[j].LastMessageID
		})

		for _, c := range cs {
			channelTreeNode := tview.NewTreeNode(util.ChannelToString(c)).
				SetReference(c.ID)
			rootTreeNode.AddChild(channelTreeNode)
		}
	} else { // Guild
		cs := app.Session.State.Guilds[guildIdx-1].Channels
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		for _, c := range cs {
			if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) && (c.ParentID == "") {
				channelTreeNode := tview.NewTreeNode(util.ChannelToString(c)).
					SetReference(c.ID)
				rootTreeNode.AddChild(channelTreeNode)
			}
		}

	CATEGORY:
		for _, c := range cs {
			if c.Type == discordgo.ChannelTypeGuildCategory {
				for _, nestedChannel := range cs {
					if nestedChannel.ParentID == c.ID {
						channelTreeNode := tview.NewTreeNode(c.Name).
							SetReference(c.ID)
						rootTreeNode.AddChild(channelTreeNode)
						continue CATEGORY
					}
				}

				channelTreeNode := tview.NewTreeNode(c.Name).
					SetReference(c.ID)
				rootTreeNode.AddChild(channelTreeNode)
			}
		}

		for _, c := range cs {
			if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) && (c.ParentID != "") {
				var parentTreeNode *tview.TreeNode
				rootTreeNode.Walk(func(node, _ *tview.TreeNode) bool {
					if node.GetReference() == c.ParentID {
						parentTreeNode = node
						return false
					}

					return true
				})

				if parentTreeNode != nil {
					channelTreeNode := tview.NewTreeNode(util.ChannelToString(c)).
						SetReference(c.ID)
					parentTreeNode.AddChild(channelTreeNode)
				}
			}
		}
	}

	app.ChannelsTreeView.SetCurrentNode(rootTreeNode)
	app.SetFocus(app.ChannelsTreeView)
}

func onChannelsTreeViewSelected(app *App, n *tview.TreeNode) {
	c, err := app.Session.State.Channel(n.GetReference().(string))
	if err != nil {
		return
	}

	if c.Type == discordgo.ChannelTypeGuildCategory {
		n.SetExpanded(!n.IsExpanded())
		return
	}

	app.SelectedChannel = c

	app.MessagesTextView.SetTitle(util.ChannelToString(c))
	app.SetFocus(app.MessageInputField)

	go func() {
		ms, err := app.Session.ChannelMessages(c.ID, app.Config.General.FetchMessagesLimit, "", "", "")
		if err != nil {
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			app.SelectedChannel.Messages = ms
			app.MessagesTextView.Write(buildMessage(app, ms[i]))
		}

		app.MessagesTextView.ScrollToEnd()
	}()
}

func onMessagesTextViewInputCapture(app *App, e *tcell.EventKey) *tcell.EventKey {
	if app.SelectedChannel == nil {
		return nil
	}

	ms := app.SelectedChannel.Messages
	if len(ms) == 0 {
		return nil
	}

	if util.HasKeybinding(app.Config.Keybindings.SelectPreviousMessage, e.Name()) {
		if len(app.MessagesTextView.GetHighlights()) == 0 {
			app.SelectedMessage = len(ms) - 1
		} else {
			app.SelectedMessage--
			if app.SelectedMessage < 0 {
				app.SelectedMessage = 0
			}
		}

		app.MessagesTextView.
			Highlight(ms[app.SelectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(app.Config.Keybindings.SelectNextMessage, e.Name()) {
		if len(app.MessagesTextView.GetHighlights()) == 0 {
			app.SelectedMessage = len(ms) - 1
		} else {
			app.SelectedMessage++
			if app.SelectedMessage >= len(ms) {
				app.SelectedMessage = len(ms) - 1
			}
		}

		app.MessagesTextView.
			Highlight(ms[app.SelectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(app.Config.Keybindings.SelectFirstMessage, e.Name()) {
		app.SelectedMessage = 0
		app.MessagesTextView.
			Highlight(ms[app.SelectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(app.Config.Keybindings.SelectLastMessage, e.Name()) {
		app.SelectedMessage = len(ms) - 1
		app.MessagesTextView.
			Highlight(ms[app.SelectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(app.Config.Keybindings.FocusMessageActionsList, e.Name()) {
		messageActionsList := tview.NewList()

		hs := app.MessagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(app.SelectedChannel.Messages, hs[0])
		if m == nil {
			return nil
		}

		if util.HasPermission(app.Session.State, app.SelectedChannel.ID, discordgo.PermissionManageMessages) || m.Author.ID == app.Session.State.User.ID {
			messageActionsList.AddItem("Edit", "", 'e', nil)
		}

		if util.HasPermission(app.Session.State, app.SelectedChannel.ID, discordgo.PermissionSendMessages) {
			messageActionsList.AddItem("Reply", "", 'r', nil)
			messageActionsList.AddItem("Mention Reply", "", 'R', nil)
		}

		if m.ReferencedMessage != nil {
			messageActionsList.AddItem("Select Reply", "", 'r', nil)
		}

		messageActionsList.
			ShowSecondaryText(false).
			AddItem("Copy Content", "", 'c', nil).
			AddItem("Copy ID", "", 'i', nil).
			SetDoneFunc(func() {
				app.SetRoot(app.MainFlex, true)
			}).
			SetSelectedFunc(func(_ int, mainText string, _ string, _ rune) {
				onMessageActionsListSelected(app, mainText, m)
			}).
			SetTitle("Press the Escape key to close").
			SetBorder(true)

		app.SetRoot(messageActionsList, true)
	}

	return e
}

func onMessageActionsListSelected(app *App, mainText string, m *discordgo.Message) {
	switch mainText {
	case "Copy Content":
		if err := clipboard.WriteAll(m.Content); err != nil {
			return
		}
	case "Copy ID":
		if err := clipboard.WriteAll(m.ID); err != nil {
			return
		}
	case "Reply":
		app.SetFocus(app.MessageInputField)
		app.MessageInputField.SetTitle("Replying to " + m.Author.String())
	case "Mention Reply":
		app.SetFocus(app.MessageInputField)
		app.MessageInputField.SetTitle("[@] Replying to " + m.Author.String())
	case "Select Reply":
		app.SelectedMessage, _ = util.FindMessageByID(app.SelectedChannel.Messages, m.ReferencedMessage.ID)
		app.MessagesTextView.
			Highlight(m.ReferencedMessage.ID).
			ScrollToHighlight()
	}

	app.SetRoot(app.MainFlex, true)
}

func onMessageInputFieldInputCapture(app *App, e *tcell.EventKey) *tcell.EventKey {
	// The default global navigation shortcut for guilds list is Alt+<rune>.
	if e.Modifiers() == tcell.ModAlt {
		return nil
	}

	switch e.Key() {
	case tcell.KeyEnter:
		if app.SelectedChannel == nil {
			return nil
		}

		t := strings.TrimSpace(app.MessageInputField.GetText())
		if t == "" {
			return nil
		}

		if len(app.MessagesTextView.GetHighlights()) != 0 {
			_, m := util.FindMessageByID(app.SelectedChannel.Messages, app.MessagesTextView.GetHighlights()[0])
			d := &discordgo.MessageSend{
				Content:         t,
				Reference:       m.Reference(),
				AllowedMentions: &discordgo.MessageAllowedMentions{RepliedUser: false},
			}
			if strings.HasPrefix(app.MessageInputField.GetTitle(), "[@]") {
				d.AllowedMentions.RepliedUser = true
			} else {
				d.AllowedMentions.RepliedUser = false
			}

			go app.Session.ChannelMessageSendComplex(m.ChannelID, d)

			app.SelectedMessage = -1
			app.MessagesTextView.Highlight()

			app.MessageInputField.SetTitle("")
		} else {
			go app.Session.ChannelMessageSend(app.SelectedChannel.ID, t)
		}

		app.MessageInputField.SetText("")
		return nil
	case tcell.KeyCtrlV:
		text, _ := clipboard.ReadAll()
		text = app.MessageInputField.GetText() + text
		app.MessageInputField.SetText(text)
		return nil
	case tcell.KeyEscape:
		app.MessageInputField.SetText("")
		app.MessageInputField.SetTitle("")

		app.SelectedMessage = -1
		app.MessagesTextView.Highlight()
		return nil
	}

	return e
}
