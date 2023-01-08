package main

import (
	"bytes"
	"log"
	"time"

	"github.com/ayn2op/discordo/discordmd"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const replyIndicator = '╭'

type MessagesText struct {
	*tview.TextView

	buffer          bytes.Buffer
	selectedMessage *discord.Message
}

func newMessagesText() *MessagesText {
	mt := &MessagesText{
		TextView: tview.NewTextView(),
	}

	mt.SetDynamicColors(true)
	mt.SetRegions(true)
	mt.SetWordWrap(true)
	mt.ScrollToEnd()
	mt.SetHighlightedFunc(mt.onHighlighted)
	mt.SetInputCapture(mt.onInputCapture)

	mt.SetBackgroundColor(tcell.GetColor(cfg.Theme.MessagesText.BackgroundColor))

	mt.SetTitle("Messages")
	mt.SetTitleColor(tcell.GetColor(cfg.Theme.MessagesText.TitleColor))
	mt.SetTitleAlign(tview.AlignLeft)

	padding := cfg.Theme.MessagesText.BorderPadding
	mt.SetBorder(cfg.Theme.MessagesText.Border)
	mt.SetBorderPadding(padding[0], padding[1], padding[2], padding[3])

	return mt
}

func (mt *MessagesText) reset() {
	messagesText.selectedMessage = nil

	mt.SetTitle("")
	mt.Clear()
	mt.Highlight()
}

func (mt *MessagesText) newMessage(m *discord.Message) error {
	mt.buffer.Reset()

	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		// Region tags are square brackets that contain a region ID in double quotes
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		mt.buffer.WriteString(`["`)
		mt.buffer.WriteString(m.ID.String())
		mt.buffer.WriteString(`"]`)

		if m.ReferencedMessage != nil {
			mt.buffer.WriteString("[::d] ")
			mt.buffer.WriteRune(replyIndicator)
			mt.buffer.WriteByte(' ')

			// Author
			mt.newAuthor(m.ReferencedMessage)

			// Content
			mt.newContent(m.ReferencedMessage)

			mt.buffer.WriteString("[::-]\n")
		}

		if cfg.Timestamps {
			// Timestamps
			mt.newTimestamp(m)
		}

		// Author
		mt.newAuthor(m)

		// Content
		mt.newContent(m)

		// Attachments
		mt.newAttachments(m)

		// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
		mt.buffer.WriteString(`[""]`)
		mt.buffer.WriteByte('\n')
	}

	_, err := mt.buffer.WriteTo(mt)
	return err
}

func (mt *MessagesText) newAuthor(m *discord.Message) {
	mt.buffer.WriteByte('[')
	mt.buffer.WriteString(cfg.Theme.MessagesText.AuthorColor)
	mt.buffer.WriteByte(']')
	mt.buffer.WriteString(m.Author.Username)
	mt.buffer.WriteString("[-] ")
}

func (mt *MessagesText) newTimestamp(m *discord.Message) {
	mt.buffer.WriteString("[::d]")
	mt.buffer.WriteString(m.Timestamp.Format(time.Kitchen))
	mt.buffer.WriteString("[::-] ")
}

func (mt *MessagesText) newContent(m *discord.Message) {
	mt.buffer.WriteString(discordmd.Parse(tview.Escape(m.Content)))
}

func (mt *MessagesText) newAttachments(m *discord.Message) {
	for _, a := range m.Attachments {
		mt.buffer.WriteByte('\n')

		mt.buffer.WriteByte('[')
		mt.buffer.WriteString(a.Filename)
		mt.buffer.WriteString("]: ")
		mt.buffer.WriteString(a.URL)
	}
}

func (mt *MessagesText) onHighlighted(added, removed, remaining []string) {
	if len(added) == 0 {
		return
	}

	sf, err := discord.ParseSnowflake(added[0])
	if err != nil {
		log.Println(err)
		return
	}

	m, err := discordState.Cabinet.Message(guildsTree.selectedChannel.ID, discord.MessageID(sf))
	if err != nil {
		log.Println(err)
		return
	}

	mt.selectedMessage = m
}

func (mt *MessagesText) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.MessagesText.Reply:
		mt.replyAction(false)
		return nil
	case cfg.Keys.MessagesText.ReplyMention:
		mt.replyAction(true)
		return nil
	case cfg.Keys.MessagesText.Cancel:
		// TODO
		guildsTree.selectedChannel = nil

		messagesText.reset()
		messageInput.reset()
		return nil
	}

	return event
}

func (mt *MessagesText) replyAction(mention bool) {
	if mt.selectedMessage == nil {
		return
	}

	var title string
	if mention {
		title += "[@] Replying to "
	} else {
		title += "Replying to "
	}

	title += mt.selectedMessage.Author.Tag()
	messageInput.SetTitle(title)

	app.SetFocus(messageInput)
}
