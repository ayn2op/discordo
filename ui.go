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

	switch n.GetLevel() {
	case 1:
		if len(n.GetChildren()) != 0 {
			n.SetExpanded(!n.IsExpanded())
			return
		}

		if n.GetText() == "Direct Messages" {
			return
		}

		gID := n.GetReference().(string)
		g, err := session.State.Guild(gID)
		if err != nil {
			return
		}

		cs := g.Channels
		sort.Slice(cs, func(i, j int) bool {
			return cs[i].Position < cs[j].Position
		})

		// Top-level channels
		createTopLevelChannelsTreeNodes(n, cs)
		// Category channels
		createCategoryChannelsTreeNodes(n, cs)
		// Second-level channels
		createSecondLevelChannelsTreeNodes(cs)
	default:
		cID := n.GetReference().(string)
		c, err := session.State.Channel(cID)
		if err != nil {
			return
		}

		if c.Type == discordgo.ChannelTypeGuildCategory {
			n.SetExpanded(!n.IsExpanded())
		} else if c.Type == discordgo.ChannelTypeGuildNews || c.Type == discordgo.ChannelTypeGuildText {
			selectedChannel = c
			app.SetFocus(messageInputField)

			title := genChannelRepr(c)
			if c.Topic != "" {
				title += " - " + c.Topic
			}
			messagesTextView.
				Clear().
				SetTitle(title)

			go renderMessages(c.ID)
		} else if c.Type == discordgo.ChannelTypeDM || c.Type == discordgo.ChannelTypeGroupDM {
			selectedChannel = c
			app.SetFocus(messageInputField)

			messagesTextView.
				Clear().
				SetTitle(genChannelRepr(c))

			go renderMessages(c.ID)
		}
	}
}

func newTextChannelTreeNode(c *discordgo.Channel) *tview.TreeNode {
	n := tview.NewTreeNode("[::d]" + genChannelRepr(c) + "[::-]").
		SetReference(c.ID)

	return n
}

func createTopLevelChannelsTreeNodes(
	n *tview.TreeNode,
	cs []*discordgo.Channel,
) {
	for _, c := range cs {
		if (c.Type == discordgo.ChannelTypeGuildText || c.Type == discordgo.ChannelTypeGuildNews) &&
			(c.ParentID == "") {
			if p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID); err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
				continue
			}

			cn := newTextChannelTreeNode(c)
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
			if p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID); err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
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
			if p, err := session.State.UserChannelPermissions(session.State.User.ID, c.ID); err != nil || p&discordgo.PermissionViewChannel != discordgo.PermissionViewChannel {
				continue
			}

			if pn := getTreeNodeByReference(c.ParentID); pn != nil {
				cn := newTextChannelTreeNode(c)
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
