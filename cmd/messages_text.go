package cmd

import (
	"errors"
	"fmt"
	"io"
	"log/slog"
	"slices"
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

	selectedMessage int
}

func newMessagesText() *MessagesText {
	mt := &MessagesText{
		TextView: tview.NewTextView(),

		selectedMessage: -1,
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

	return mt
}

func (mt *MessagesText) drawMsgs(cID discord.ChannelID) {
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
	mainFlex.messagesText.selectedMessage = -1

	mt.SetTitle("")
	mt.Clear()
	mt.Highlight()
}

// Region tags are square brackets that contain a region ID in double quotes
// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
func (mt *MessagesText) startRegion(m discord.Message) {
	fmt.Fprintf(mt, `["%s"]`, m.ID)
}

// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
func (mt *MessagesText) endRegion() {
	fmt.Fprint(mt, `[""]`)
}

func (mt *MessagesText) createMessage(m discord.Message) {
	mt.startRegion(m)
	if cfg.HideBlockedUsers {
		isBlocked := discordState.UserIsBlocked(m.Author.ID)
		if isBlocked {
			fmt.Fprintln(mt, "[:red:b]Blocked message[:-:-]")
			mt.endRegion()
			return
		}
	}

	switch m.Type {
	case discord.ChannelPinnedMessage:
		fmt.Fprint(mt, "[" + cfg.Theme.MessagesText.ContentColor + "]" + m.Author.Username + " pinned a message" + "[-:-:-]")
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
	mt.endRegion()
}

func (mt *MessagesText) createHeader(w io.Writer, m discord.Message, isReply bool) {
	time := m.Timestamp.Time().In(time.Local).Format(cfg.TimestampsFormat)

	if cfg.Timestamps && cfg.TimestampsBeforeAuthor {
		fmt.Fprintf(w, "[::d]%s[::-] ", time)
	}

	if isReply {
		fmt.Fprintf(mt, "[::d]%s", cfg.Theme.MessagesText.ReplyIndicator)
	}

	fmt.Fprintf(w, "[%s]%s[-:-:-] ", cfg.Theme.MessagesText.AuthorColor, m.Author.Username)

	if cfg.Timestamps && !cfg.TimestampsBeforeAuthor {
		fmt.Fprintf(w, "[::d]%s[::-] ", time)
	}
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
		fmt.Fprintf(w, "[%s]: %s", a.Filename, a.URL)
	}
}

func (mt *MessagesText) getSelectedMessage() (*discord.Message, error) {
	if mt.selectedMessage == -1 {
		return nil, errors.New("no message is currently selected")
	}

	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		return nil, err
	}

	return &ms[mt.selectedMessage], nil
}

func (mt *MessagesText) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.SelectPrevious, cfg.Keys.SelectNext, cfg.Keys.SelectFirst, cfg.Keys.SelectLast, cfg.Keys.MessagesText.SelectReply:
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

	switch name {
	case cfg.Keys.SelectPrevious:
		// If no message is currently selected, select the latest message.
		if len(mt.GetHighlights()) == 0 {
			mt.selectedMessage = 0
		} else {
			if mt.selectedMessage < len(ms)-1 {
				mt.selectedMessage++
			} else {
				return
			}
		}
	case cfg.Keys.SelectNext:
		// If no message is currently selected, select the latest message.
		if len(mt.GetHighlights()) == 0 {
			mt.selectedMessage = 0
		} else {
			if mt.selectedMessage > 0 {
				mt.selectedMessage--
			} else {
				return
			}
		}
	case cfg.Keys.SelectFirst:
		mt.selectedMessage = len(ms) - 1
	case cfg.Keys.SelectLast:
		mt.selectedMessage = 0
	case cfg.Keys.MessagesText.SelectReply:
		if mt.selectedMessage == -1 {
			return
		}

		if ref := ms[mt.selectedMessage].ReferencedMessage; ref != nil {
			for i, m := range ms {
				if ref.ID == m.ID {
					mt.selectedMessage = i
				}
			}
		} else if ref := ms[mt.selectedMessage].Reference; ref != nil {
			for i, m := range ms {
				if ref.MessageID == m.ID {
					mt.selectedMessage = i
				}
			}
		}
	}

	mt.Highlight(ms[mt.selectedMessage].ID.String())
	mt.ScrollToHighlight()
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
	mainFlex.messageInput.replyMessageIdx = mt.selectedMessage
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
