package main

import (
	"bytes"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type MessagesText struct {
	*tview.TextView

	buffer bytes.Buffer
}

func newMessagesText() *MessagesText {
	mt := &MessagesText{
		TextView: tview.NewTextView(),
	}

	mt.SetDynamicColors(true)
	mt.SetRegions(true)
	mt.SetWordWrap(true)

	mt.SetTitle("Messages")
	mt.SetBorder(true)
	mt.SetBorderPadding(cfg.BorderPadding())

	return mt
}

func (mt *MessagesText) newMessage(m *discord.Message) error {
	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		// Region tags are square brackets that contain a region ID in double quotes
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		mt.buffer.WriteString(`["`)
		mt.buffer.WriteString(m.ID.String())
		mt.buffer.WriteString(`"]`)

		if m.ReferencedMessage != nil {
			mt.buffer.WriteString("> [::d]")

			// Author
			mt.newAuthor(m.ReferencedMessage)
			// Content
			mt.newContent(m.ReferencedMessage)

			mt.buffer.WriteString("[::-]")
			mt.buffer.WriteByte('\n')
		}

		// Author
		mt.newAuthor(m)
		// Timestamps
		mt.newTimestamp(m)
		// Content
		mt.buffer.WriteByte('\n')
		mt.newContent(m)

		// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
		mt.buffer.WriteString(`[""]`)
	}

	mt.buffer.WriteString("\n\n")

	_, err := mt.buffer.WriteTo(mt)
	return err
}

func (mt *MessagesText) newAuthor(m *discord.Message) {
	mt.buffer.WriteString("[blue]")
	mt.buffer.WriteString(m.Author.Username)
	mt.buffer.WriteString("[-] ")
}

func (mt *MessagesText) newTimestamp(m *discord.Message) {
	mt.buffer.WriteString("[::d]")
	mt.buffer.WriteString(m.Timestamp.Format(time.Kitchen))
	mt.buffer.WriteString("[-:-:-] ")
}

func (mt *MessagesText) newContent(m *discord.Message) {
	mt.buffer.WriteString(m.Content)
}
