package main

import (
	"sort"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayntgl/discordgo"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	selectedChannel *discordgo.Channel
	selectedMessage int = -1
)

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if hasKeybinding(conf.Keybindings.FocusChannelsTree, e.Name()) {
		app.SetFocus(channelsTree)
		return nil
	} else if hasKeybinding(conf.Keybindings.FocusMessagesView, e.Name()) {
		app.SetFocus(messagesView)
		return nil
	} else if hasKeybinding(conf.Keybindings.FocusMessageInputField, e.Name()) {
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
					if channelIsUnread(session.State, c) {
						tag = "[::b]"
					}

					cn := tview.NewTreeNode(tag + channelToString(c) + "[::-]").
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
				createTopLevelChannelsNodes(channelsTree, session.State, n, g.Channels)
				// Category channels
				createCategoryChannelsNodes(channelsTree, session.State, n, g.Channels)
				// Second-level channels
				createSecondLevelChannelsNodes(channelsTree, session.State, g.Channels)
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
			title := channelToString(c)
			if c.Topic != "" {
				title += " - " + c.Topic
			}

			messagesView.SetTitle(title)
		case discordgo.ChannelTypeDM, discordgo.ChannelTypeGroupDM:
			messagesView.SetTitle(channelToString(c))
		}

		if strings.HasPrefix(n.GetText(), "[::b]") {
			n.SetText("[::d]" + channelToString(c) + "[::-]")
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

			if len(ms) != 0 && channelIsUnread(session.State, c) {
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
			if !hasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			n.AddChild(createChannelNode(s, c))
			continue
		}
	}
}

// createCategoryChannelsNodes builds and creates `*tview.TreeNode`s for category (type: GUILD_CATEGORY) channels. If the client user does not have the VIEW_CHANNEL permission for a channel, the channel is excluded from the parent.
func createCategoryChannelsNodes(treeView *tview.TreeView, s *discordgo.State, n *tview.TreeNode, cs []*discordgo.Channel) {
CategoryLoop:
	for _, c := range cs {
		if c.Type == discordgo.ChannelTypeGuildCategory {
			if !hasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			for _, child := range cs {
				if child.ParentID == c.ID {
					n.AddChild(createChannelNode(s, c))
					continue CategoryLoop
				}
			}

			n.AddChild(createChannelNode(s, c))
		}
	}
}

// createSecondLevelChannelsNodes builds and creates `*tview.TreeNode`s for second-level (channels that have a non-empty parent ID and of type GUILD_TEXT, GUILD_NEWS) channels. If the client user does not have the VIEW_CHANNEL permission for a channel, the channel is excluded from the parent.
func createSecondLevelChannelsNodes(treeView *tview.TreeView, s *discordgo.State, cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID != "") {
			if !hasPermission(s, c.ID, discordgo.PermissionViewChannel) {
				continue
			}

			pn := getTreeNodeByReference(treeView, c.ParentID)
			if pn != nil {
				pn.AddChild(createChannelNode(s, c))
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

	if hasKeybinding(conf.Keybindings.SelectPreviousMessage, e.Name()) {
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
	} else if hasKeybinding(conf.Keybindings.SelectNextMessage, e.Name()) {
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
	} else if hasKeybinding(conf.Keybindings.SelectFirstMessage, e.Name()) {
		selectedMessage = 0
		messagesView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if hasKeybinding(conf.Keybindings.SelectLastMessage, e.Name()) {
		selectedMessage = len(ms) - 1
		messagesView.
			Highlight(ms[selectedMessage].ID).
			ScrollToHighlight()
		return nil
	} else if hasKeybinding(conf.Keybindings.SelectMessageReference, e.Name()) {
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := findMessageByID(selectedChannel.Messages, hs[0])
		if m.ReferencedMessage != nil {
			selectedMessage, _ = findMessageByID(selectedChannel.Messages, m.ReferencedMessage.ID)
			messagesView.
				Highlight(m.ReferencedMessage.ID).
				ScrollToHighlight()
		}

		return nil
	} else if hasKeybinding(conf.Keybindings.ReplySelectedMessage, e.Name()) {
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := findMessageByID(selectedChannel.Messages, hs[0])
		messageInputField.SetTitle("Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	} else if hasKeybinding(conf.Keybindings.MentionReplySelectedMessage, e.Name()) {
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := findMessageByID(selectedChannel.Messages, hs[0])
		messageInputField.SetTitle("[@] Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	} else if hasKeybinding(conf.Keybindings.CopySelectedMessage, e.Name()) {
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := findMessageByID(selectedChannel.Messages, hs[0])
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

// getTreeNodeByReference walks the root `*TreeNode` of the given `*TreeView` *treeView* and returns the TreeNode whose reference is equal to the given reference *r*. If the `*TreeNode` is not found, `nil` is returned instead.
func getTreeNodeByReference(treeView *tview.TreeView, r interface{}) (mn *tview.TreeNode) {
	treeView.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}

// createChannelNode builds (encorporates unread channels in bold tag, otherwise dim, etc.) and returns a node according to the type of the given channel *c*.
func createChannelNode(s *discordgo.State, c *discordgo.Channel) *tview.TreeNode {
	var cn *tview.TreeNode
	switch c.Type {
	case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews:
		tag := "[::d]"
		if channelIsUnread(s, c) {
			tag = "[::b]"
		}

		cn = tview.NewTreeNode(tag + channelToString(c) + "[::-]").
			SetReference(c.ID)
	case discordgo.ChannelTypeGuildCategory:
		cn = tview.NewTreeNode(c.Name).
			SetReference(c.ID)
	}

	return cn
}

// hasPermission returns a boolean that indicates whether the client user has the given permission *p* in the given channel ID *cID*.
func hasPermission(s *discordgo.State, cID string, p int64) bool {
	perm, err := s.UserChannelPermissions(s.User.ID, cID)
	if err != nil {
		return false
	}

	return perm&p == p
}

func hasKeybinding(sl []string, s string) bool {
	for _, str := range sl {
		if str == s {
			return true
		}
	}

	return false
}
