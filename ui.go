package main

import (
	"sort"
	"strings"

	"github.com/ayntgl/discordgo"
	"github.com/ayntgl/discordo/clipboard"
	"github.com/ayntgl/discordo/util"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	selectedChannel *discordgo.Channel
	selectedMessage int = -1
)

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case conf.Keybindings.FocusChannelsTree:
		app.SetFocus(channelsTree)
		return nil
	case conf.Keybindings.FocusMessagesView:
		app.SetFocus(messagesView)
		return nil
	case conf.Keybindings.FocusMessageInputField:
		app.SetFocus(messageInputField)
		return nil
	}

	return e
}

func onChannelsTreeSelected(n *tview.TreeNode) {
	selectedChannel = nil
	selectedMessage = 0
	messagesView.
		Clear().
		SetTitle("")
	messageInputField.SetText("")
	// Unhighlight the already-highlighted regions.
	messagesView.Highlight()

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
				util.CreateTopLevelChannelsNodes(channelsTree, session.State, n, g.Channels)
				// Category channels
				util.CreateCategoryChannelsNodes(channelsTree, session.State, n, g.Channels)
				// Second-level channels
				util.CreateSecondLevelChannelsNodes(channelsTree, session.State, g.Channels)
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

			messagesView.SetTitle(title)
		case discordgo.ChannelTypeDM, discordgo.ChannelTypeGroupDM:
			messagesView.SetTitle(util.ChannelToString(c))
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
				messagesView.Write(buildMessage(ms[i]))
			}
			// Scroll to the end of the text after the messages have been written to the TextView.
			messagesView.ScrollToEnd()

			if len(ms) != 0 && util.ChannelIsUnread(session.State, c) {
				session.ChannelMessageAck(c.ID, c.LastMessageID, "")
			}
		}()
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

	switch e.Name() {
	case conf.Keybindings.SelectPreviousMessage:
		if len(messagesView.GetHighlights()) == 0 {
			selectedMessage = len(ms) - 1
		} else {
			selectedMessage--
			if selectedMessage < 0 {
				selectedMessage = 0
			}
		}

		messagesView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	case conf.Keybindings.SelectNextMessage:
		if len(messagesView.GetHighlights()) == 0 {
			selectedMessage = len(ms) - 1
		} else {
			selectedMessage++
			if selectedMessage >= len(ms) {
				selectedMessage = len(ms) - 1
			}
		}

		messagesView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	case conf.Keybindings.SelectFirstMessage:
		selectedMessage = 0
		messagesView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	case conf.Keybindings.SelectLastMessage:
		selectedMessage = len(ms) - 1
		messagesView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	case conf.Keybindings.SelectMessageReference:
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		if m.ReferencedMessage != nil {
			selectedMessage, _ = util.FindMessageByID(selectedChannel.Messages, m.ReferencedMessage.ID)
			messagesView.
				Highlight(m.ReferencedMessage.ID).
				ScrollToHighlight()
		}

		return nil
	case conf.Keybindings.ReplySelectedMessage:
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		messageInputField.SetTitle("Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	case conf.Keybindings.MentionReplySelectedMessage:
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		messageInputField.SetTitle("[@] Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	case conf.Keybindings.CopySelectedMessage:
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		err := clipboard.Write([]byte(m.Content))
		if err != nil {
			return nil
		}
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

		if len(messagesView.GetHighlights()) != 0 {
			m := selectedChannel.Messages[selectedMessage]
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
			messageInputField.SetTitle("")
		} else {
			go session.ChannelMessageSend(selectedChannel.ID, t)
		}

		messageInputField.SetText("")
		return nil
	case tcell.KeyCtrlV:
		b, err := clipboard.Read()
		if err != nil {
			return nil
		}

		messageInputField.SetText(messageInputField.GetText() + string(b))
		return nil
	case tcell.KeyEscape:
		messageInputField.SetText("")
		messageInputField.SetTitle("")

		selectedMessage = -1
		messagesView.Highlight()
		return nil
	}

	return e
}
