package main

import (
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayntgl/discordgo"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

var (
	app               *tview.Application
	loginForm         *tview.Form
	channelsTree      *tview.TreeView
	messagesTextView  *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex
)

func newApplication() *tview.Application {
	a := tview.NewApplication()
	a.
		EnableMouse(conf.Mouse).
		SetInputCapture(onAppInputCapture)

	return a
}

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case conf.Keybindings.GuildsTreeViewFocus:
		app.SetFocus(channelsTree)
	case conf.Keybindings.MessagesTextViewFocus:
		app.SetFocus(messagesTextView)
	case conf.Keybindings.MessageInputFieldFocus:
		app.SetFocus(messageInputField)
	}

	return e
}

func newChannelsTree() *tview.TreeView {
	channelsTree := tview.NewTreeView()
	channelsTree.
		SetSelectedFunc(onChannelsTreeSelected).
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return channelsTree
}

func onChannelsTreeSelected(n *tview.TreeNode) {
	selectedChannel = nil
	selectedMessage = nil
	messagesTextView.
		Clear().
		SetTitle("")
	messageInputField.SetText("")
	// Unhighlight the already-highlighted regions.
	messagesTextView.Highlight()

	if len(n.GetChildren()) != 0 || n.GetText() == "Direct Messages" {
		n.SetExpanded(!n.IsExpanded())
	} else {
		cID := n.GetReference().(string)
		c, err := session.State.Channel(cID)
		if err != nil {
			return
		}

		selectedChannel = c
		app.SetFocus(messageInputField)

		switch c.Type {
		case discordgo.ChannelTypeGuildText, discordgo.ChannelTypeGuildNews:
			title := generateChannelRepr(c)
			if c.Topic != "" {
				title += " - " + c.Topic
			}

			messagesTextView.SetTitle(title)
		case discordgo.ChannelTypeDM, discordgo.ChannelTypeGroupDM:
			messagesTextView.SetTitle(generateChannelRepr(c))
		}

		if strings.HasPrefix(n.GetText(), "[::b]") {
			n.SetText("[::d]" + generateChannelRepr(c) + "[::-]")
		}

		messagesTextView.Clear()

		ms, err := session.ChannelMessages(cID, conf.GetMessagesLimit, "", "", "")
		if err != nil {
			return
		}

		for i := len(ms) - 1; i >= 0; i-- {
			selectedChannel.Messages = append(selectedChannel.Messages, ms[i])
			go renderMessage(ms[i])
		}

		if len(ms) != 0 && isUnread(c) {
			go session.ChannelMessageAck(c.ID, c.LastMessageID, "")
		}
	}
}

func createTopLevelChannelsTreeNodes(
	n *tview.TreeNode,
	cs []*discordgo.Channel,
) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID == "") {
			p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID)
			if err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
				continue
			}

			var tag string
			if isUnread(c) {
				tag = "[::b]"
			} else {
				tag = "[::d]"
			}

			cn := tview.NewTreeNode(tag + generateChannelRepr(c) + "[::-]").
				SetReference(c.ID)
			n.AddChild(cn)
			continue
		}
	}
}

func createCategoryChannelsTreeNodes(
	n *tview.TreeNode,
	cs []*discordgo.Channel,
) {
CategoryLoop:
	for _, c := range cs {
		if c.Type == discordgo.ChannelTypeGuildCategory {
			p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID)
			if err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
				continue
			}

			for _, child := range cs {
				if child.ParentID == c.ID {
					cn := tview.NewTreeNode(c.Name).
						SetReference(c.ID)
					n.AddChild(cn)
					continue CategoryLoop
				}
			}

			cn := tview.NewTreeNode(c.Name).
				SetReference(c.ID)
			n.AddChild(cn)
		}
	}
}

func createSecondLevelChannelsTreeNodes(cs []*discordgo.Channel) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID != "") {
			p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID)
			if err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
				continue
			}

			var tag string
			if isUnread(c) {
				tag = "[::b]"
			} else {
				tag = "[::d]"
			}

			pn := getTreeNodeByReference(c.ParentID)
			if pn != nil {
				cn := tview.NewTreeNode(tag + generateChannelRepr(c) + "[::-]").
					SetReference(c.ID)
				pn.AddChild(cn)
			}
		}
	}
}

func getTreeNodeByReference(r interface{}) (mn *tview.TreeNode) {
	channelsTree.GetRoot().Walk(func(n, _ *tview.TreeNode) bool {
		if n.GetReference() == r {
			mn = n
			return false
		}

		return true
	})

	return
}

func newMessagesTextView() *tview.TextView {
	w := tview.NewTextView()
	w.
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetInputCapture(onMessagesTextViewInputCapture).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return w
}

func onMessagesTextViewInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if selectedChannel == nil {
		return nil
	}

	switch e.Name() {
	case conf.Keybindings.MessagesTextViewSelectPrevious:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			messagesTextView.
				Highlight(ms[len(ms)-1].ID).
				ScrollToHighlight()
		} else {
			idx, _ := findByMessageID(ms, hs[0])
			if idx == -1 || idx == 0 {
				return nil
			}

			messagesTextView.
				Highlight(ms[idx-1].ID).
				ScrollToHighlight()
		}

		return nil
	case conf.Keybindings.MessagesTextViewSelectNext:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			messagesTextView.
				Highlight(ms[len(ms)-1].ID).
				ScrollToHighlight()
		} else {
			idx, _ := findByMessageID(ms, hs[0])
			if idx == -1 || idx == len(ms)-1 {
				return nil
			}

			messagesTextView.
				Highlight(ms[idx+1].ID).
				ScrollToHighlight()
		}

		return nil
	case conf.Keybindings.MessagesTextViewSelectFirst:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		messagesTextView.
			Highlight(ms[0].ID).
			ScrollToHighlight()
	case conf.Keybindings.MessagesTextViewSelectLast:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		messagesTextView.
			Highlight(ms[len(ms)-1].ID).
			ScrollToHighlight()
	case conf.Keybindings.MessagesTextViewReplySelected:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, selectedMessage = findByMessageID(ms, hs[0])
		messageInputField.SetTitle(
			"Replying to " + selectedMessage.Author.Username,
		)
		app.SetFocus(messageInputField)
	case conf.Keybindings.MessagesTextViewMentionReplySelected:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		hs := messagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, selectedMessage = findByMessageID(ms, hs[0])
		messageInputField.SetTitle("[@] Repling to " + selectedMessage.Author.Username)
		app.SetFocus(messageInputField)
	}

	return e
}

func newMessageInputField() *tview.InputField {
	w := tview.NewInputField()
	w.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetInputCapture(onMessageInputFieldInputCapture).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return w
}

func onMessageInputFieldInputCapture(e *tcell.EventKey) *tcell.EventKey {
	// If the "Alt" modifier key is pressed, do not handle the event.
	if e.Modifiers() == tcell.ModAlt {
		return nil
	}

	switch e.Key() {
	case tcell.KeyEnter:
		if selectedChannel == nil {
			return nil
		}

		t := strings.TrimSpace(messageInputField.GetText())
		if t == "" {
			return nil
		}

		if selectedMessage != nil {
			d := &discordgo.MessageSend{
				Content:         t,
				Reference:       selectedMessage.Reference(),
				AllowedMentions: &discordgo.MessageAllowedMentions{RepliedUser: false},
			}
			if strings.HasPrefix(messageInputField.GetTitle(), "[@]") {
				d.AllowedMentions.RepliedUser = true
			} else {
				d.AllowedMentions.RepliedUser = false
			}

			go session.ChannelMessageSendComplex(selectedMessage.ChannelID, d)
			messageInputField.SetTitle("")
			selectedMessage = nil
		} else {
			go session.ChannelMessageSend(selectedChannel.ID, t)
		}

		messageInputField.SetText("")
	case tcell.KeyCtrlV:
		text, _ := clipboard.ReadAll()
		text = messageInputField.GetText() + text
		messageInputField.SetText(text)
	case tcell.KeyEscape:
		messageInputField.SetTitle("")
		selectedMessage = nil
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
