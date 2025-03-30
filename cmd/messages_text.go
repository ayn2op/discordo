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
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/renderer"
	"github.com/yuin/goldmark/text"
)

type MessagesText struct {
	*tview.TextView
	cfg               *config.Config
	app               *tview.Application
	selectedMessageID discord.MessageID
}

func newMessagesText(app *tview.Application, cfg *config.Config) *MessagesText {
	mt := &MessagesText{
		TextView: tview.NewTextView(),
		cfg:      cfg,
		app:      app,
	}

	t := cfg.Theme
	mt.
		SetDynamicColors(true).
		SetRegions(true).
		SetWordWrap(true).
		ScrollToEnd().
		SetTextColor(tcell.GetColor(t.MessagesText.ContentColor)).
		SetHighlightedFunc(mt.onHighlighted).
		SetChangedFunc(func() {
			app.Draw()
		})

	b := t.Border
	p := b.Padding
	mt.
		SetInputCapture(mt.onInputCapture).
		SetTitle("Messages").
		SetTitleAlign(tview.AlignLeft).
		SetBorder(b.Enabled).
		SetBorderPadding(p[0], p[1], p[2], p[3]).
		SetFocusFunc(func() {
			mt.SetBorderColor(tcell.GetColor(b.ActiveColor))
			mt.SetTitleColor(tcell.GetColor(t.ActiveTitleColor))
		}).
		SetBlurFunc(func() {
			mt.SetBorderColor(tcell.GetColor(b.Color))
			mt.SetTitleColor(tcell.GetColor(t.TitleColor))
		})

	markdown.DefaultRenderer.AddOptions(
		renderer.WithOption("emojiColor", t.MessagesText.EmojiColor),
		renderer.WithOption("linkColor", t.MessagesText.LinkColor),
	)

	return mt
}

func (mt *MessagesText) drawMsgs(cID discord.ChannelID) {
	ms, err := discordState.Messages(cID, uint(mt.cfg.MessagesLimit))
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", cID)
		return
	}

	for _, m := range slices.Backward(ms) {
		app.messagesText.createMessage(m)
	}
}

func (mt *MessagesText) reset() {
	app.messagesText.selectedMessageID = 0

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

func (mt *MessagesText) createMessage(m discord.Message) {
	mt.startRegion(m.ID)
	defer mt.endRegion()

	if mt.cfg.HideBlockedUsers {
		isBlocked := discordState.UserIsBlocked(m.Author.ID)
		if isBlocked {
			fmt.Fprintln(mt, "[:red:b]Blocked message[:-:-]")
			return
		}
	}

	switch m.Type {
	case discord.ChannelPinnedMessage:
		fmt.Fprint(mt, "["+mt.cfg.Theme.MessagesText.ContentColor+"]"+m.Author.Username+" pinned a message"+"[-:-:-]")
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
	if mt.cfg.Timestamps {
		time := m.Timestamp.Time().In(time.Local).Format(mt.cfg.TimestampsFormat)
		fmt.Fprintf(w, "[::d]%s[::-] ", time)
	}

	if isReply {
		fmt.Fprintf(mt, "[::d]%s", mt.cfg.Theme.MessagesText.ReplyIndicator)
	}

	fmt.Fprintf(w, "[%s]%s[-:-:-] ", mt.cfg.Theme.MessagesText.AuthorColor, m.Author.Username)
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
		if mt.cfg.ShowAttachmentLinks {
			fmt.Fprintf(w, "[%s][%s]:\n%s[-]", mt.cfg.Theme.MessagesText.AttachmentColor, a.Filename, a.URL)
		} else {
			fmt.Fprintf(w, "[%s][%s][-]", mt.cfg.Theme.MessagesText.AttachmentColor, a.Filename)
		}
	}
}

func (mt *MessagesText) getSelectedMessage() (*discord.Message, error) {
	if !mt.selectedMessageID.IsValid() {
		return nil, errors.New("no message is currently selected")
	}

	msg, err := discordState.Cabinet.Message(app.guildsTree.selectedChannelID, mt.selectedMessageID)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve selected message: %w", err)
	}

	return msg, nil
}

func (mt *MessagesText) getSelectedMessageIndex() (int, error) {
	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
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
	case mt.cfg.Keys.MessagesText.SelectPrevious, mt.cfg.Keys.MessagesText.SelectNext, mt.cfg.Keys.MessagesText.SelectFirst, mt.cfg.Keys.MessagesText.SelectLast, mt.cfg.Keys.MessagesText.SelectReply, mt.cfg.Keys.MessagesText.SelectPin:
		mt._select(event.Name())
	case mt.cfg.Keys.MessagesText.YankID:
		mt.yankID()
	case mt.cfg.Keys.MessagesText.YankContent:
		mt.yankContent()
	case mt.cfg.Keys.MessagesText.YankURL:
		mt.yankURL()
	case mt.cfg.Keys.MessagesText.Open:
		mt.open()
	case mt.cfg.Keys.MessagesText.Reply:
		mt.reply(false)
	case mt.cfg.Keys.MessagesText.ReplyMention:
		mt.reply(true)
	case mt.cfg.Keys.MessagesText.Delete:
		mt.delete()
	}

	return nil
}

func (mt *MessagesText) _select(name string) {
	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
	if err != nil {
		slog.Error("failed to get messages", "err", err, "channel_id", app.guildsTree.selectedChannelID)
		return
	}

	messageIdx, err := mt.getSelectedMessageIndex()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	switch name {
	case mt.cfg.Keys.MessagesText.SelectPrevious:
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
	case mt.cfg.Keys.MessagesText.SelectNext:
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
	case mt.cfg.Keys.MessagesText.SelectFirst:
		mt.selectedMessageID = ms[len(ms)-1].ID
	case mt.cfg.Keys.MessagesText.SelectLast:
		mt.selectedMessageID = ms[0].ID
	case mt.cfg.Keys.MessagesText.SelectReply:
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
	case mt.cfg.Keys.MessagesText.SelectPin:
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

func (mt *MessagesText) yankID() {
	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err := clipboard.WriteAll(msg.ID.String()); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *MessagesText) yankContent() {
	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err = clipboard.WriteAll(msg.Content); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *MessagesText) yankURL() {
	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	if err = clipboard.WriteAll(msg.URL()); err != nil {
		slog.Error("failed to write to clipboard", "err", err)
	}
}

func (mt *MessagesText) open() {
	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	var urls []string
	if msg.Content != "" {
		urls = extractURLs(msg.Content)
	}

	if len(urls) == 0 && len(msg.Attachments) == 0 {
		return
	}

	if len(urls)+len(msg.Attachments) == 1 {
		if len(urls) == 1 {
			go openURL(urls[0])
		} else {
			go openURL(msg.Attachments[0].URL)
		}
	} else {
		mt.showUrlSelector(urls, msg.Attachments)
	}
}

func extractURLs(content string) []string {
	src := []byte(content)
	node := parser.NewParser(
		parser.WithBlockParsers(discordmd.BlockParsers()...),
		parser.WithInlineParsers(discordmd.InlineParserWithLink()...),
	).Parse(text.NewReader(src))

	var urls []string
	ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n := n.(type) {
			case *ast.AutoLink:
				urls = append(urls, string(n.URL(src)))
			case *ast.Link:
				urls = append(urls, string(n.Destination))
			}
		}

		return ast.WalkContinue, nil
	})
	return urls
}

func (mt *MessagesText) showUrlSelector(urls []string, attachments []discord.Attachment) {
	done := func() {
		app.pages.RemovePage("list").SwitchToPage("flex")
		app.SetFocus(app.messagesText)
	}

	list := tview.NewList().
		SetWrapAround(true).
		SetHighlightFullLine(true).
		ShowSecondaryText(false).
		SetDoneFunc(done)

	b := mt.cfg.Theme.Border
	p := b.Padding
	list.
		SetBorder(b.Enabled).
		SetBorderColor(tcell.GetColor(b.Color)).
		SetBorderPadding(p[0], p[1], p[2], p[3])

	list.
		SetInputCapture(func(event *tcell.EventKey) *tcell.EventKey {
			switch event.Name() {
			case mt.cfg.Keys.MessagesText.SelectPrevious:
				return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
			case mt.cfg.Keys.MessagesText.SelectNext:
				return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
			case mt.cfg.Keys.MessagesText.SelectFirst:
				return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
			case mt.cfg.Keys.MessagesText.SelectLast:
				return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
			}

			return event
		})

	for i, a := range attachments {
		attachment := a
		list.AddItem(a.Filename, "", rune('a'+i), func() {
			go openURL(attachment.URL)
			done()
		})
	}

	for i, url := range urls {
		urlCopy := url
		list.AddItem(url, "", rune('1'+i), func() {
			go openURL(urlCopy)
			done()
		})
	}

	app.pages.
		AddAndSwitchToPage("list", ui.Centered(list, 0, 0), true).
		ShowPage("flex")
}

func openURL(url string) {
	if err := open.Start(url); err != nil {
		slog.Error("failed to open URL", "err", err, "url", url)
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
	app.messageInput.SetTitle(title)
	app.messageInput.replyMessageID = mt.selectedMessageID
	mt.app.SetFocus(app.messageInput)
}

func (mt *MessagesText) delete() {
	msg, err := mt.getSelectedMessage()
	if err != nil {
		slog.Error("failed to get selected message", "err", err)
		return
	}

	clientID := discordState.Ready().User.ID
	if msg.GuildID.IsValid() {
		ps, err := discordState.Permissions(app.guildsTree.selectedChannelID, discordState.Ready().User.ID)
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

	if err := discordState.DeleteMessage(app.guildsTree.selectedChannelID, msg.ID, ""); err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID)
		return
	}

	if err := discordState.MessageRemove(app.guildsTree.selectedChannelID, msg.ID); err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", app.guildsTree.selectedChannelID, "message_id", msg.ID)
		return
	}

	ms, err := discordState.Cabinet.Messages(app.guildsTree.selectedChannelID)
	if err != nil {
		slog.Error("failed to delete message", "err", err, "channel_id", app.guildsTree.selectedChannelID)
		return
	}

	mt.Clear()

	for _, m := range slices.Backward(ms) {
		app.messagesText.createMessage(m)
	}
}
