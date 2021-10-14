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
	messagesView      *tview.TextView
	messageInputField *tview.InputField
	mainFlex          *tview.Flex
)

func newApp() *tview.Application {
	a := tview.NewApplication()
	a.
		EnableMouse(conf.Mouse).
		SetInputCapture(onAppInputCapture)

	return a
}

func onAppInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case conf.Keybindings.ChannelsTree.Focus:
		app.SetFocus(channelsTree)
	case conf.Keybindings.MessagesView.Focus:
		app.SetFocus(messagesView)
	case conf.Keybindings.MessageInputField.Focus:
		app.SetFocus(messageInputField)
	}

	return e
}

func newChannelsTree() *tview.TreeView {
	treeView := tview.NewTreeView()
	treeView.
		SetSelectedFunc(onChannelsTreeSelected).
		SetTopLevel(1).
		SetRoot(tview.NewTreeNode("")).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0)

	return treeView
}

func onChannelsTreeSelected(n *tview.TreeNode) {
	selectedChannel = nil
	selectedMessage = nil
	messagesView.
		Clear().
		SetTitle("")
	messageInputField.SetText("")
	// Unhighlight the already-highlighted regions.
	messagesView.Highlight()

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

			messagesView.SetTitle(title)
		case discordgo.ChannelTypeDM, discordgo.ChannelTypeGroupDM:
			messagesView.SetTitle(generateChannelRepr(c))
		}

		if strings.HasPrefix(n.GetText(), "[::b]") {
			n.SetText("[::d]" + generateChannelRepr(c) + "[::-]")
		}

		go func() {
			ms, err := session.ChannelMessages(cID, conf.GetMessagesLimit, "", "", "")
			if err != nil {
				return
			}

			for i := len(ms) - 1; i >= 0; i-- {
				selectedChannel.Messages = append(selectedChannel.Messages, ms[i])
				renderMessage(ms[i])
			}

			if len(ms) != 0 && isUnread(c) {
				session.ChannelMessageAck(c.ID, c.LastMessageID, "")
			}
		}()
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

func newMessagesView() *tview.TextView {
	textView := tview.NewTextView()
	textView.
		SetRegions(true).
		SetDynamicColors(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetChangedFunc(func() {
			app.Draw()
		}).
		SetInputCapture(onMessagesViewInputCapture).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return textView
}

func onMessagesViewInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if selectedChannel == nil {
		return nil
	}

	switch e.Name() {
	case conf.Keybindings.MessagesView.SelectPrevious:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			messagesView.
				Highlight(ms[len(ms)-1].ID).
				ScrollToHighlight()
		} else {
			idx, _ := findByMessageID(ms, hs[0])
			if idx == -1 || idx == 0 {
				return nil
			}

			messagesView.
				Highlight(ms[idx-1].ID).
				ScrollToHighlight()
		}

		return nil
	case conf.Keybindings.MessagesView.SelectNext:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			messagesView.
				Highlight(ms[len(ms)-1].ID).
				ScrollToHighlight()
		} else {
			idx, _ := findByMessageID(ms, hs[0])
			if idx == -1 || idx == len(ms)-1 {
				return nil
			}

			messagesView.
				Highlight(ms[idx+1].ID).
				ScrollToHighlight()
		}

		return nil
	case conf.Keybindings.MessagesView.SelectFirst:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		messagesView.
			Highlight(ms[0].ID).
			ScrollToHighlight()
	case conf.Keybindings.MessagesView.SelectLast:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		messagesView.
			Highlight(ms[len(ms)-1].ID).
			ScrollToHighlight()
	case conf.Keybindings.MessagesView.Reply:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, selectedMessage = findByMessageID(ms, hs[0])
		messageInputField.SetTitle(
			"Replying to " + selectedMessage.Author.Username,
		)
		app.SetFocus(messageInputField)
	case conf.Keybindings.MessagesView.ReplyMention:
		ms := selectedChannel.Messages
		if len(ms) == 0 {
			return nil
		}

		hs := messagesView.GetHighlights()
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
	inputField := tview.NewInputField()
	inputField.
		SetPlaceholder("Message...").
		SetPlaceholderTextColor(tcell.ColorWhite).
		SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor).
		SetInputCapture(onMessageInputFieldInputCapture).
		SetBorder(true).
		SetBorderPadding(0, 0, 1, 0).
		SetTitleAlign(tview.AlignLeft)

	return inputField
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
