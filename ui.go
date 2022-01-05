package main

import (
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/util"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if util.HasKeybinding(conf.Keybindings.FocusChannelsTree, e.Name()) {
		app.SetFocus(channelsTreeView)
		return nil
	} else if util.HasKeybinding(conf.Keybindings.FocusMessagesView, e.Name()) {
		app.SetFocus(messagesTextView)
		return nil
	} else if util.HasKeybinding(conf.Keybindings.FocusMessageInputField, e.Name()) {
		app.SetFocus(messageInputField)
		return nil
	}

	return e
}

func onChannelsTreeSelected(n *tview.TreeNode) {
	selectedChannel = nil
	selectedMessage = 0
	messagesTextView.
		Clear().
		SetTitle("")
	messageInputField.SetText("")
	// Unhighlight the already-highlighted regions.
	messagesTextView.Highlight()

	id := n.GetReference()
	switch n.GetLevel() {
	case 1: // Guilds or Direct Messages
		if len(n.GetChildren()) == 0 {
			// If the reference of the selected `*TreeNode` is `nil`, it is the direct messages `*TreeNode`.
			if id == nil {
				cs := session.State.PrivateChannels
				sort.Slice(cs, func(i, j int) bool {
					return cs[i].LastMessageID > cs[j].LastMessageID
				})

				for _, c := range cs {
					tag := "[::d]"
					if util.ChannelIsUnread(session.State, c) {
						tag = "[::b]"
					}

					cn := tview.NewTreeNode(tag + util.ChannelToString(c) + "[::-]").
						SetReference(c.ID).
						Collapse()
					n.AddChild(cn)
				}
			} else {
				g, err := session.State.Guild(id.(string))
				if err != nil {
					return
				}

				sort.Slice(g.Channels, func(i, j int) bool {
					return g.Channels[i].Position < g.Channels[j].Position
				})

				// Top-level channels
				createTopLevelChannelsNodes(channelsTreeView, session.State, n, g.Channels)
				// Category channels
				createCategoryChannelsNodes(channelsTreeView, session.State, n, g.Channels)
				// Second-level channels
				createSecondLevelChannelsNodes(channelsTreeView, session.State, g.Channels)
			}
		}

		n.SetExpanded(!n.IsExpanded())
	default: // Channels
		c, err := session.State.Channel(id.(string))
		if err != nil {
			return
		}

		selectedChannel = c
		app.SetFocus(messageInputField)

		switch c.Type {
		case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews:
			title := util.ChannelToString(c)
			if c.Topic != "" {
				title += " - " + c.Topic
			}

			messagesTextView.SetTitle(title)
		case discordgo.ChannelTypeDM, discordgo.ChannelTypeGroupDM:
			messagesTextView.SetTitle(util.ChannelToString(c))
		}

		if strings.HasPrefix(n.GetText(), "[::b]") {
			n.SetText("[::d]" + util.ChannelToString(c) + "[::-]")
		}

		go func() {
			ms, err := session.ChannelMessages(c.ID, conf.GetMessagesLimit, "", "", "")
			if err != nil {
				return
			}

			for i := len(ms) - 1; i >= 0; i-- {
				selectedChannel.Messages = append(selectedChannel.Messages, ms[i])
				messagesTextView.Write(buildMessage(ms[i]))
			}
			// Scroll to the end of the text after the messages have been written to the TextView.
			messagesTextView.ScrollToEnd()

			if len(ms) != 0 && util.ChannelIsUnread(session.State, c) {
				session.ChannelMessageAck(c.ID, c.LastMessageID, "")
			}
		}()
	}
}

// createTopLevelChannelsNodes builds and creates `*tview.TreeNode`s for top-level (channels that have an empty parent ID and of type GUILD_TEXT, GUILD_NEWS) channels. If the client user does not have the VIEW_CHANNEL permission for a channel, the channel is excluded from the parent.
func createTopLevelChannelsNodes(treeView *tview.TreeView, s *discordgo.State, n *tview.TreeNode, cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID == "") {
			if !util.HasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			n.AddChild(util.CreateChannelNode(s, c))
			continue
		}
	}
}

// createCategoryChannelsNodes builds and creates `*tview.TreeNode`s for category (type: GUILD_CATEGORY) channels. If the client user does not have the VIEW_CHANNEL permission for a channel, the channel is excluded from the parent.
func createCategoryChannelsNodes(treeView *tview.TreeView, s *discordgo.State, n *tview.TreeNode, cs []*discordgo.Channel) {
CategoryLoop:
	for _, c := range cs {
		if c.Type == discordgo.ChannelTypeGuildCategory {
			if !util.HasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			for _, child := range cs {
				if child.ParentID == c.ID {
					n.AddChild(util.CreateChannelNode(s, c))
					continue CategoryLoop
				}
			}

			n.AddChild(util.CreateChannelNode(s, c))
		}
	}
}

// createSecondLevelChannelsNodes builds and creates `*tview.TreeNode`s for second-level (channels that have a non-empty parent ID and of type GUILD_TEXT, GUILD_NEWS) channels. If the client user does not have the VIEW_CHANNEL permission for a channel, the channel is excluded from the parent.
func createSecondLevelChannelsNodes(treeView *tview.TreeView, s *discordgo.State, cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID != "") {
			if !util.HasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			pn := util.GetNodeByReference(treeView, c.ParentID)
			if pn != nil {
				pn.AddChild(util.CreateChannelNode(s, c))
			}
		}
	}
}

func onMessagesViewInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if selectedChannel == nil {
		return nil
	}

	ms := selectedChannel.Messages
	if len(ms) == 0 {
		return nil
	}

	if util.HasKeybinding(conf.Keybindings.SelectPreviousMessage, e.Name()) {
		if len(messagesTextView.GetHighlights()) == 0 {
			selectedMessage = len(ms) - 1
		} else {
			selectedMessage--
			if selectedMessage < 0 {
				selectedMessage = 0
			}
		}

		messagesTextView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(conf.Keybindings.SelectNextMessage, e.Name()) {
		if len(messagesTextView.GetHighlights()) == 0 {
			selectedMessage = len(ms) - 1
		} else {
			selectedMessage++
			if selectedMessage >= len(ms) {
				selectedMessage = len(ms) - 1
			}
		}

		messagesTextView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(conf.Keybindings.SelectFirstMessage, e.Name()) {
		selectedMessage = 0
		messagesTextView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(conf.Keybindings.SelectLastMessage, e.Name()) {
		selectedMessage = len(ms) - 1
		messagesTextView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(conf.Keybindings.SelectMessageReference, e.Name()) {
		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		if m.ReferencedMessage != nil {
			selectedMessage, _ = util.FindMessageByID(selectedChannel.Messages, m.ReferencedMessage.ID)
			messagesTextView.
				Highlight(m.ReferencedMessage.ID).
				ScrollToHighlight()
		}

		return nil
	} else if util.HasKeybinding(conf.Keybindings.ReplySelectedMessage, e.Name()) {
		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		messageInputField.SetTitle("Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	} else if util.HasKeybinding(conf.Keybindings.MentionReplySelectedMessage, e.Name()) {
		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		messageInputField.SetTitle("[@] Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	} else if util.HasKeybinding(conf.Keybindings.CopySelectedMessage, e.Name()) {
		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		err := clipboard.WriteAll(m.Content)
		if err != nil {
			return nil
		}

		return nil
	}

	return e
}

func onMessageInputFieldInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Key() {
	case tcell.KeyEnter:
		if selectedChannel == nil {
			return nil
		}

		t := strings.TrimSpace(messageInputField.GetText())
		if t == "" {
			return nil
		}

		if len(messagesTextView.GetHighlights()) != 0 {
			_, m := util.FindMessageByID(selectedChannel.Messages, messagesTextView.GetHighlights()[0])
			d := &discordgo.MessageSend{
				Content:         t,
				Reference:       m.Reference(),
				AllowedMentions: &discordgo.MessageAllowedMentions{RepliedUser: false},
			}
			if strings.HasPrefix(messageInputField.GetTitle(), "[@]") {
				d.AllowedMentions.RepliedUser = true
			} else {
				d.AllowedMentions.RepliedUser = false
			}

			go session.ChannelMessageSendComplex(m.ChannelID, d)

			selectedMessage = -1
			messagesTextView.Highlight()

			messageInputField.SetTitle("")
		} else {
			go session.ChannelMessageSend(selectedChannel.ID, t)
		}

		messageInputField.SetText("")
		return nil
	case tcell.KeyCtrlV:
		text, _ := clipboard.ReadAll()
		text = messageInputField.GetText() + text
		messageInputField.SetText(text)
		return nil
	case tcell.KeyEscape:
		messageInputField.SetText("")
		messageInputField.SetTitle("")

		selectedMessage = -1
		messagesTextView.Highlight()
		return nil
	}

	return e
}
