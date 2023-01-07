package main

import (
	"fmt"
	"log"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessagesText struct {
	*tview.TextView

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

	mt.SetTitle("Messages")
	mt.SetTitleAlign(tview.AlignLeft)

	padding := cfg.Theme.MessagesText.BorderPadding
	mt.SetBorder(cfg.Theme.MessagesText.Border)
	mt.SetBorderPadding(padding[0], padding[1], padding[2], padding[3])

	return mt
}

func (mt *MessagesText) newMessage(m *discord.Message) error {
	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		// Region tags are square brackets that contain a region ID in double quotes
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		fmt.Fprintf(mt, `["%s"]`, m.ID)

		if m.ReferencedMessage != nil {
			fmt.Fprint(mt, "[::d] ╭ ")

			// Author
			mt.newAuthor(m.ReferencedMessage)

			// Content
			mt.newContent(m.ReferencedMessage)

			fmt.Fprint(mt, "[::-]")
			fmt.Fprintln(mt)
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
		fmt.Fprint(mt, `[""]`)
	}

	fmt.Fprintln(mt)
	return nil
}

func (mt *MessagesText) newAuthor(m *discord.Message) {
	fmt.Fprintf(mt, "[blue]%s[-] ", m.Author.Username)
}

func (mt *MessagesText) newTimestamp(m *discord.Message) {
	fmt.Fprintf(mt, "[::d]%s[::-] ", m.Timestamp.Format(time.Kitchen))
}

func (mt *MessagesText) newContent(m *discord.Message) {
	fmt.Fprint(mt, tview.Escape(m.Content))
}

func (mt *MessagesText) newAttachments(m *discord.Message) {
	for _, a := range m.Attachments {
		fmt.Fprintf(mt, "\n[%s]: %s", a.Filename, a.URL)
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
