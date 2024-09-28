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
	*tview.TextView

	selectedMessageID discord.MessageID
}

type NewMessagesText struct {
	*MessagesText
	*tview.Box

	selectedMessageID discord.MessageID
	messageBoxes []*MessageBox
	init bool
}

func newNewMessagesText() *NewMessagesText{
	mt := &NewMessagesText{
		Box: tview.NewBox(),
		init: true,
	}

	mt.SetBorder(true)
	mt.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))

	mt.SetDrawFunc(func(screen tcell.Screen, x int, y int, width int, height int) (int, int, int, int) {
		if mt.init {
			prevLineCount := 0
			for _, m := range mt.messageBoxes {
				m.SetText("")
				m.SetRect(x+1, y+1+prevLineCount, width-2, (height-2-prevLineCount))
				src := []byte(m.Content)
				ast := discordmd.ParseWithMessage(src, *discordState.Cabinet, m.Message, false)
				// send parsed text to message box...
				markdown.DefaultRenderer.Render(m.TextView, src, ast)
				// ...so it can rewrite and call Draw immediately
				m.SetText(m.GetText(false)).Draw(screen)

				prevLineCount += m.getLineCount()
			}
			mt.init = false 
		} else {
			for _, m := range mt.messageBoxes {
				m.SetText(m.GetText(false)).Draw(screen)
			}
		}
		
		return x, y, width, height
  	})

	mt.SetFocusFunc(func(){mt.init = true})
	mt.SetBlurFunc(func(){mt.init = true})

	markdown.DefaultRenderer.AddOptions(
		renderer.WithOption("emojiColor", cfg.Theme.MessagesText.EmojiColor),
		renderer.WithOption("linkColor", cfg.Theme.MessagesText.LinkColor),
	)
	return mt
}

func newMessagesText() *MessagesText {
	mt := &MessagesText{
		TextView: tview.NewTextView(),
	}

	mt.SetDynamicColors(true)
	mt.SetRegions(true)
	mt.SetWordWrap(true)
	mt.SetInputCapture(mt.onInputCapture)
	mt.ScrollToEnd()
	mt.SetChangedFunc(func() {
		app.Draw()
	})

	mt.SetTextColor(tcell.GetColor(cfg.Theme.MessagesText.ContentColor))
	mt.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))

	mt.SetTitle("Messages")
	mt.SetTitleColor(tcell.GetColor(cfg.Theme.TitleColor))
	mt.SetTitleAlign(tview.AlignLeft)

	p := cfg.Theme.BorderPadding
	mt.SetBorder(cfg.Theme.Border)
	mt.SetBorderColor(tcell.GetColor(cfg.Theme.BorderColor))
	mt.SetBorderPadding(p[0], p[1], p[2], p[3])

	markdown.DefaultRenderer.AddOptions(
		renderer.WithOption("emojiColor", cfg.Theme.MessagesText.EmojiColor),
		renderer.WithOption("linkColor", cfg.Theme.MessagesText.LinkColor),
	)

	mt.SetHighlightedFunc(mt.onHighlighted)

	return mt
}

func (mt *NewMessagesText) drawMsgs(cID discord.ChannelID) {
	mt.messageBoxes = nil
	ms, err := discordState.Messages(cID, uint(cfg.MessagesLimit))
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", cID)
		return
	}

	for _, m := range slices.Backward(ms) {
		mainFlex.messagesText.createMessage(m)
	}

	mt.init = true
}

func (mt *NewMessagesText) reset() {
	mainFlex.messagesText.selectedMessageID = 0

	mt.SetTitle("")
	mt.Clear()
	mt.Highlight()
}

// Region tags are square brackets that contain a region ID in double quotes
// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
func (mt *MessagesText) startRegion(msgID discord.MessageID) {
	fmt.Fprintf(mt, `["%s"]`, msgID)
}

// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
func (mt *MessagesText) endRegion() {
	fmt.Fprint(mt, `[""]`)
}

func (mt *NewMessagesText) createMessage(m discord.Message) {
	mb := newMessageBox()
	mb.Message = &m

	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
	}

	mt.messageBoxes = append(mt.messageBoxes, mb)
}

func (mt *MessagesText) oldCreateMessage(m discord.Message) {
	mt.startRegion(m.ID)
	defer mt.endRegion()

	if cfg.HideBlockedUsers {
		isBlocked := discordState.UserIsBlocked(m.Author.ID)
		if isBlocked {
			fmt.Fprintln(mt, "[:red:b]Blocked message[:-:-]")
			return
		}
	}

	switch m.Type {
	case discord.ChannelPinnedMessage:
		fmt.Fprint(mt, "["+cfg.Theme.MessagesText.ContentColor+"]"+m.Author.Username+" pinned a message"+"[-:-:-]")
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		if m.ReferencedMessage != nil {
			mt.createHeader(mt, *m.ReferencedMessage, true)
			mt.createBody(mt, *m.ReferencedMessage, true)

			fmt.Fprint(mt, "[::-]\n")
		}

		mt.createHeader(mt, m, false)
		mt.createBody(mt, m, false)
		mt.createFooter(mt, m)
	default:
		mt.createHeader(mt, m, false)
	}

	fmt.Fprintln(mt)
}

func (mt *MessagesText) createHeader(w io.Writer, m discord.Message, isReply bool) {
	if cfg.Timestamps {
		time := m.Timestamp.Time().In(time.Local).Format(cfg.TimestampsFormat)
		fmt.Fprintf(w, "[::d]%s[::-] ", time)
	}

	if isReply {
		fmt.Fprintf(mt, "[::d]%s", cfg.Theme.MessagesText.ReplyIndicator)
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
		if len(mt.GetHighlights()) == 0 {
			mt.selectedMessageID = ms[0].ID
		} else {
			if messageIdx < len(ms)-1 {
				mt.selectedMessageID = ms[messageIdx+1].ID
			} else {
				return
			}
		}
	case cfg.Keys.SelectNext:
		// If no message is currently selected, select the latest message.
		if len(mt.GetHighlights()) == 0 {
			mt.selectedMessageID = ms[0].ID
		} else {
			if messageIdx > 0 {
				mt.selectedMessageID = ms[messageIdx-1].ID
			} else {
				return
			}
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

	mt.Highlight(mt.selectedMessageID.String())
	mt.ScrollToHighlight()
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

	mt.Clear()

	for _, m := range slices.Backward(ms) {
		mainFlex.messagesText.createMessage(m)
	}
}
