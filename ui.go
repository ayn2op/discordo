package main

import (
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

func newApp() *tview.Application {
	a := tview.NewApplication()
	a.
		EnableMouse(conf.Mouse).
		SetInputCapture(onAppInputCapture)

	return a
}

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
	selectedMessage = 0
	messagesView.
		Clear().
		SetTitle("")
	messageInputField.SetText("")
	// Unhighlight the already-highlighted regions.
	messagesView.Highlight()

	ref := n.GetReference()
	// If the node's reference is nil, the selected node is a guild or direct messages node; expand or collapse accordingly.
	if ref == nil {
		n.SetExpanded(!n.IsExpanded())
	} else {
		c, err := session.State.Channel(ref.(string))
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

			if len(ms) != 0 && isUnread(c) {
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
	case conf.Keybindings.ReplySelectedMessage:
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		m := findByMessageID(hs[0])
		messageInputField.SetTitle("Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	case conf.Keybindings.MentionReplySelectedMessage:
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		m := findByMessageID(hs[0])
		messageInputField.SetTitle("[@] Replying to " + m.Author.String())
		app.SetFocus(messageInputField)
		return nil
	case conf.Keybindings.CopySelectedMessage:
		hs := messagesView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		m := findByMessageID(hs[0])
		err := clipboard.WriteAll(m.Content)
		if err != nil {
			return nil
		}
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
