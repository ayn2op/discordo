package cmd

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
	"strconv"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
	"github.com/yuin/goldmark/renderer"
)

type MessagesText struct {
	*tview.Box

	selectedMessageID discord.MessageID
	messageBoxes []*MessageBox
}

func newMessagesText() *MessagesText{
	mt := &MessagesText{
		Box: tview.NewBox(),
	}

	mt.SetBorder(true)
	mt.SetTitle("Messages")
	mt.SetTitleAlign(tview.AlignLeft)
	mt.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))
	mt.Box.SetInputCapture(mt.onInputCapture)

	mt.SetDrawFunc(func(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
		messageIdx, err := mt.getSelectedMessageIndex()
		if err != nil {
			slog.Error("failed to get selected message", "err", err)
		}
		messageIdx = min(messageIdx, len(mt.messageBoxes)-1)

		// force bottom-first if no message is selected
		if messageIdx == -1 {
			mt.renderMessagesBottomFirst(x, y, width, height, screen)
			mt.renderTopBorder(x, y, width, height, screen)
			return x, y, width, height
		}

		// Position messages without any scrolling offset
		prevLineCount := 0
		for _, m := range mt.messageBoxes {
			mY := y+1+prevLineCount
			mH := height-2-prevLineCount
			m.SetRect(x+1, mY, width-2, mH)
			prevLineCount += m.getLineCount(width-1)
		}

		// Get scrolling offset based on selected message
		_, selectedY, _, _ := mt.messageBoxes[len(mt.messageBoxes)-1-messageIdx].GetRect()
		scrollOffset := 20
		scrollY := -(selectedY - (y+1) - scrollOffset)
		
		// Get first and last message coordinates to determine the render options
		_, firstY, _, _ := mt.messageBoxes[0].GetRect()
		_, lastY, _, _ := mt.messageBoxes[len(mt.messageBoxes)-1].GetRect()
		firstY += scrollY
		lastY += scrollY
		lastLineCount := mt.messageBoxes[len(mt.messageBoxes)-1].getLineCount(width-1)

		if firstY > y+1 {
			// top-first rendering
			prevLineCount = 0
			for _, m := range mt.messageBoxes {
				m.SetRect(x+1, y+1+prevLineCount, width-2, height-2-prevLineCount)
				prevLineCount += m.getLineCount(width-1)
				m.Render(mt.selectedMessageID == m.ID, screen)
				if y+1+prevLineCount > height-2 {
					break
				}
			}
		} else if lastY+lastLineCount < height-1 {
			mt.renderMessagesBottomFirst(x, y, width, height, screen)
			mt.renderTopBorder(x, y, width, height, screen)
		} else {
			// in-between rendering 
			for _, m := range mt.messageBoxes {
				mX, mY, mW, mH := m.GetRect()
				mY += scrollY
				mH -= scrollY
				m.SetRect(mX, mY, mW, mH)
				m.Render(mt.selectedMessageID == m.ID, screen)
			}
			mt.renderTopBorder(x, y, width, height, screen)
		}

		return x, y, width, height
  	})

	markdown.DefaultRenderer.AddOptions(
		renderer.WithOption("emojiColor", cfg.Theme.MessagesText.EmojiColor),
		renderer.WithOption("linkColor", cfg.Theme.MessagesText.LinkColor),
	)
	return mt
}

func (mt *MessagesText) renderMessagesBottomFirst(x int, y int, width int, height int, screen tcell.Screen) {
	prevLineCount := 0
	for _, m := range slices.Backward(mt.messageBoxes) {
		lineCount := m.getLineCount(width-1)
		prevLineCount += lineCount
		m.SetRect(x+1, height-1-prevLineCount, width-2, lineCount)
		m.Render(mt.selectedMessageID == m.ID, screen)
		if height-1-prevLineCount + lineCount < y+1 {
			break
		}
	}
}

func (mt *MessagesText) renderTopBorder(x int, y int, width int, height int, screen tcell.Screen) {
	topLine := mt.GetTitle()
	for i := 0; i < width-2 - len(mt.GetTitle()); i++ {
		if mt.HasFocus() {
			topLine += "═"
		} else {
			topLine += "─"
		}
	}
	tview.PrintSimple(screen, topLine, x+1, y)
}

func (mt *MessagesText) drawMsgs(cID discord.ChannelID) {
	mt.messageBoxes = nil
	ms, err := discordState.Messages(cID, uint(cfg.MessagesLimit))
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", cID)
		return
	}

	for _, m := range slices.Backward(ms) {
		mainFlex.messagesText.createMessage(m)
	}
}

func (mt *MessagesText) reset() {
	mainFlex.messagesText.selectedMessageID = 0
	mt.SetTitle("")
}

func (mt *MessagesText) createMessage(m discord.Message) {
	mb := newMessageBox()
	mb.Message = &m
	fmt.Fprintf(mb, `["msg"]`)

	switch m.Type {
	case discord.ChannelPinnedMessage:
			fmt.Fprint(mb, "["+cfg.Theme.MessagesText.ContentColor+"]"+m.Author.Username+" pinned a message"+"[-:-:-]")
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		if m.ReferencedMessage != nil {
			mt.createHeader(mb, *m.ReferencedMessage, true)
			mt.createBody(mb, *m.ReferencedMessage, true)

			fmt.Fprint(mb, "[::-]\n")
		}

		mt.createHeader(mb, m, false)
		mt.createBody(mb, m, false)
		mt.createFooter(mb, m)
	}

	mt.messageBoxes = append(mt.messageBoxes, mb)
}

func (mt *MessagesText) createHeader(w io.Writer, m discord.Message, isReply bool) {
	if cfg.Timestamps {
		time := m.Timestamp.Time().In(time.Local).Format(cfg.TimestampsFormat)
		fmt.Fprintf(w, "[::d]%s[::-] ", time)
	}

	if isReply {
		fmt.Fprintf(w, "[::d]%s", cfg.Theme.MessagesText.ReplyIndicator)
	}

	fmt.Fprintf(w, "[%s]%s[-:-:-] ", cfg.Theme.MessagesText.AuthorColor, m.Author.Username)
}

func (mt *MessagesText) createBody(w io.Writer, m discord.Message, isReply bool) {
	if isReply {
		fmt.Fprint(w, "[::d]")
	}

	src := []byte(m.Content)
	ast := discordmd.ParseWithMessage(src, *discordState.Cabinet, &m, false)
	markdown.DefaultRenderer.Render(w, src, ast)

	if isReply {
		fmt.Fprint(w, "[::-]")
	}
}

func (mt *MessagesText) createFooter(w io.Writer, m discord.Message) {
	for _, a := range m.Attachments {
		fmt.Fprintln(w)
		if cfg.ShowAttachmentLinks {
			fmt.Fprintf(w, "[%s][%s]:\n%s[-]", cfg.Theme.MessagesText.AttachmentColor, a.Filename, a.URL)
		} else {
			fmt.Fprintf(w, "[%s][%s][-]", cfg.Theme.MessagesText.AttachmentColor, a.Filename)
		}
	}
}

func (mt *MessagesText) getSelectedMessage() (*discord.Message, error) {
	if !mt.selectedMessageID.IsValid() {
		return nil, errors.New("no message is currently selected")
	}

	msg, err := discordState.Cabinet.Message(mainFlex.guildsTree.selectedChannelID, mt.selectedMessageID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve selected message: %w", err)
	}

	return msg, nil
}

func (mt *MessagesText) getSelectedMessageIndex() (int, error) {
	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		return -1, err
	}

	for idx, m := range ms {
		for m.ID == mt.selectedMessageID {
			return idx, nil
		}
	}

	return -1, nil
}

func (mt *MessagesText) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.SelectPrevious, cfg.Keys.SelectNext, cfg.Keys.SelectFirst, cfg.Keys.SelectLast, cfg.Keys.MessagesText.SelectReply, cfg.Keys.MessagesText.SelectPin:
		mt._select(event.Name())
		return nil
	case cfg.Keys.MessagesText.Yank:
		mt.yank()
		return nil
	case cfg.Keys.MessagesText.Open:
		mt.open()
		return nil
	case cfg.Keys.MessagesText.Reply:
		mt.reply(false)
		return nil
	case cfg.Keys.MessagesText.ReplyMention:
		mt.reply(true)
		return nil
	case cfg.Keys.MessagesText.Delete:
		mt.delete()
		return nil
	}

	return nil
}

func (mt *MessagesText) _select(name string) {
	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", mainFlex.guildsTree.selectedChannelID)
		return
	}

	messageIdx, err := mt.getSelectedMessageIndex()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	switch name {
	case cfg.Keys.SelectPrevious:
		// If no message is currently selected, select the latest message.
		if messageIdx == -1 {
			mt.selectedMessageID = ms[0].ID
		} else if messageIdx < len(ms)-1 {
			mt.selectedMessageID = ms[messageIdx+1].ID
		}
	case cfg.Keys.SelectNext:
		// If no message is currently selected, select the latest message.
		if messageIdx == -1 { 
			mt.selectedMessageID = ms[0].ID
		} else if messageIdx > 0 {
			mt.selectedMessageID = ms[messageIdx-1].ID
		}
	case cfg.Keys.SelectFirst:
		mt.selectedMessageID = ms[len(ms)-1].ID
	case cfg.Keys.SelectLast:
		mt.selectedMessageID = ms[0].ID
	case cfg.Keys.MessagesText.SelectReply:
		if mt.selectedMessageID == 0 {
			return
		}

		if ref := ms[messageIdx].ReferencedMessage; ref != nil {
			for _, m := range ms {
				if ref.ID == m.ID {
					mt.selectedMessageID = m.ID
				}
			}
		}
	case cfg.Keys.MessagesText.SelectPin:
		if ref := ms[messageIdx].Reference; ref != nil {
			for _, m := range ms {
				if ref.MessageID == m.ID {
					mt.selectedMessageID = m.ID
				}
			}
		}
	}
}

func (mt *MessagesText) onHighlighted(added, removed, remaining []string) {
	if len(added) > 0 {
		mID, err := strconv.ParseInt(added[0], 10, 64)
		if err != nil {
			slog.Error("Failed to parse region id as int to use as message id.", "err", err)
			return
		}
		mt.selectedMessageID = discord.MessageID(mID)
	}
}

func (mt *MessagesText) yank() {
	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	err = clipboard.WriteAll(msg.Content)
	if err != nil {
		slog.Error("failed to write to clipboard", "err", err)
		return
	}
}

func (mt *MessagesText) open() {
	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	attachments := msg.Attachments
	if len(attachments) == 0 {
		return
	}

	for _, a := range attachments {
		go func() {
			if err := open.Start(a.URL); err != nil {
				slog.Error("failed to open URL", "err", err, "url", a.URL)
			}
		}()
	}
}

func (mt *MessagesText) reply(mention bool) {
	var title string
	if mention {
		title += "[@] Replying to "
	} else {
		title += "Replying to "
	}

	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	title += msg.Author.Tag()
	mainFlex.messageInput.SetTitle(title)
	mainFlex.messageInput.replyMessageID = mt.selectedMessageID
	app.SetFocus(mainFlex.messageInput)
}

func (mt *MessagesText) delete() {
	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	clientID := discordState.Ready().User.ID
	if msg.GuildID.IsValid() {
		ps, err := discordState.Permissions(mainFlex.guildsTree.selectedChannelID, discordState.Ready().User.ID)
		if err != nil {
			return
		}

		if msg.Author.ID != clientID && !ps.Has(discord.PermissionManageMessages) {
			return
		}
	} else {
		if msg.Author.ID != clientID {
			return
		}
	}

	if err := discordState.DeleteMessage(mainFlex.guildsTree.selectedChannelID, msg.ID, ""); err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", mainFlex.guildsTree.selectedChannelID, "message_id", msg.ID)
		return
	}

	if err := discordState.MessageRemove(mainFlex.guildsTree.selectedChannelID, msg.ID); err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", mainFlex.guildsTree.selectedChannelID, "message_id", msg.ID)
		return
	}

	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", mainFlex.guildsTree.selectedChannelID)
		return
	}

	for _, m := range slices.Backward(ms) {
		mainFlex.messagesText.createMessage(m)
	}
}
