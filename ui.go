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

var (
	selectedChannel *discordgo.Channel
	selectedMessage int = -1
)

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if util.HasKeybinding(conf.Keybindings.FocusChannelsTree, e.Name()) {
		app.SetFocus(channelsTree)
		return nil
	} else if util.HasKeybinding(conf.Keybindings.FocusMessagesView, e.Name()) {
		app.SetFocus(messagesView)
		return nil
	} else if util.HasKeybinding(conf.Keybindings.FocusMessageInputField, e.Name()) {
		app.SetFocus(messageInputField)
		return nil
	}

	return e
}

func newChannelsTree() *tview.TreeView {
	treeView := tview.NewTreeView()
	treeView.
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetTitle("Channels").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return treeView
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

func newMessagesView() *tview.TextView {
	textView := tview.NewTextView()
	textView.
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return textView
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
	} else if util.HasKeybinding(conf.Keybindings.SelectNextMessage, e.Name()) {
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
	} else if util.HasKeybinding(conf.Keybindings.SelectFirstMessage, e.Name()) {
		selectedMessage = 0
		messagesView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(conf.Keybindings.SelectLastMessage, e.Name()) {
		selectedMessage = len(ms) - 1
		messagesView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if util.HasKeybinding(conf.Keybindings.SelectMessageReference, e.Name()) {
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
	} else if util.HasKeybinding(conf.Keybindings.ReplySelectedMessage, e.Name()) {
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		messageInputField.SetTitle("Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	} else if util.HasKeybinding(conf.Keybindings.MentionReplySelectedMessage, e.Name()) {
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := util.FindMessageByID(selectedChannel.Messages, hs[0])
		messageInputField.SetTitle("[@] Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	} else if util.HasKeybinding(conf.Keybindings.CopySelectedMessage, e.Name()) {
		hs := messagesView.GetHighlights()
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

func newMessageInputField() *tview.InputField {
	inputField := tview.NewInputField()
	inputField.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return inputField
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

			selectedMessage = -1
			messagesView.Highlight()

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
		messagesView.Highlight()
		return nil
	}

	return e
}

func newLoginForm(onLoginFormLoginButtonSelected func(), mfa bool) *tview.Form {
	w := tview.NewForm()
	w.
		AddButton("Login", onLoginFormLoginButtonSelected).
		SetButtonsAlign(tview.AlignCenter).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	if mfa {
		w.AddPasswordField("Code", "", 0, 0, nil)
	} else {
		w.
			AddInputField("Email", "", 0, nil, nil).
			AddPasswordField("Password", "", 0, 0, nil)
	}

	return w
}
