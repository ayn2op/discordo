package main

import (
	"bytes"
	"log"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/discordmd"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

const replyIndicator = '╭'

type MessagesText struct {
	*tview.TextView

	selectedMessage int
	buf             bytes.Buffer
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

	mt.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))

	mt.SetTitle("Messages")
	mt.SetTitleColor(tcell.GetColor(cfg.Theme.TitleColor))
	mt.SetTitleAlign(tview.AlignLeft)

	p := cfg.Theme.BorderPadding
	mt.SetBorder(cfg.Theme.Border)
	mt.SetBorderColor(tcell.GetColor(cfg.Theme.BorderColor))
	mt.SetBorderPadding(p[0], p[1], p[2], p[3])

	return mt
}

func (mt *MessagesText) reset() {
	messagesText.selectedMessage = -1

	mt.SetTitle("")
	mt.Clear()
	mt.Highlight()
}

func (mt *MessagesText) createMessage(m *discord.Message) error {
	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		// Region tags are square brackets that contain a region ID in double quotes
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		mt.buf.WriteString(`["`)
		mt.buf.WriteString(m.ID.String())
		mt.buf.WriteString(`"]`)

		if m.ReferencedMessage != nil {
			mt.buf.WriteString("[::d] ")
			mt.buf.WriteRune(replyIndicator)
			mt.buf.WriteByte(' ')

			mt.buf.WriteByte('[')
			mt.buf.WriteString(cfg.Theme.MessagesText.AuthorColor)
			mt.buf.WriteByte(']')
			mt.buf.WriteString(m.ReferencedMessage.Author.Username)
			mt.buf.WriteString("[-] ")

			mt.buf.WriteString(discordmd.Parse(tview.Escape(m.ReferencedMessage.Content)))
			mt.buf.WriteString("[::-]\n")
		}

		mt.createHeader(m)
		mt.createBody(m)
		mt.createFooter(m)

		// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
		mt.buf.WriteString(`[""]`)
		mt.buf.WriteByte('\n')
	}

	_, err := mt.buf.WriteTo(mt)
	return err
}

func (mt *MessagesText) createHeader(m *discord.Message) {
	mt.buf.WriteByte('[')
	mt.buf.WriteString(cfg.Theme.MessagesText.AuthorColor)
	mt.buf.WriteByte(']')
	mt.buf.WriteString(m.Author.Username)
	mt.buf.WriteString("[-] ")

	if cfg.Timestamps {
		mt.buf.WriteString("[::d]")
		mt.buf.WriteString(m.Timestamp.Format(time.Kitchen))
		mt.buf.WriteString("[::-] ")
	}
}

func (mt *MessagesText) createBody(m *discord.Message) {
	mt.buf.WriteString(discordmd.Parse(tview.Escape(m.Content)))
}

func (mt *MessagesText) createFooter(m *discord.Message) {
	for _, a := range m.Attachments {
		mt.buf.WriteByte('\n')

		mt.buf.WriteByte('[')
		mt.buf.WriteString(a.Filename)
		mt.buf.WriteString("]: ")
		mt.buf.WriteString(a.URL)
	}
}

func (mt *MessagesText) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.MessagesText.CopyContent:
		mt.copyContentAction()
		return nil
	case cfg.Keys.MessagesText.Reply:
		mt.replyAction(false)
		return nil
	case cfg.Keys.MessagesText.ReplyMention:
		mt.replyAction(true)
		return nil
	case cfg.Keys.MessagesText.SelectPrevious:
		mt.selectPreviousAction()
		return nil
	case cfg.Keys.MessagesText.SelectNext:
		mt.selectNextAction()
		return nil
	case cfg.Keys.MessagesText.SelectFirst:
		mt.selectFirstAction()
		return nil
	case cfg.Keys.MessagesText.SelectLast:
		mt.selectLastAction()
		return nil
	case cfg.Keys.MessagesText.SelectReply:
		mt.selectReplyAction()
		return nil
	case cfg.Keys.Cancel:
		guildsTree.selectedChannel = nil

		messagesText.reset()
		messageInput.reset()
		return nil
	}

	return event
}

func (mt *MessagesText) replyAction(mention bool) {
	if mt.selectedMessage == -1 {
		return
	}

	var title string
	if mention {
		title += "[@] Replying to "
	} else {
		title += "Replying to "
	}

	ms, err := discordState.Cabinet.Messages(guildsTree.selectedChannel.ID)
	if err != nil {
		log.Println(err)
		return
	}

	title += ms[mt.selectedMessage].Author.Tag()
	messageInput.SetTitle(title)

	app.SetFocus(messageInput)
}

func (mt *MessagesText) selectPreviousAction() {
	ms, err := discordState.Cabinet.Messages(guildsTree.selectedChannel.ID)
	if err != nil {
		log.Println(err)
		return
	}

	// If no message is currently selected, select the latest message.
	if len(mt.GetHighlights()) == 0 {
		mt.selectedMessage = 0
	} else {
		if mt.selectedMessage < len(ms)-1 {
			mt.selectedMessage++
		}
	}

	mt.Highlight(ms[mt.selectedMessage].ID.String())
	mt.ScrollToHighlight()
}

func (mt *MessagesText) selectNextAction() {
	ms, err := discordState.Cabinet.Messages(guildsTree.selectedChannel.ID)
	if err != nil {
		log.Println(err)
		return
	}

	// If no message is currently selected, select the latest message.
	if len(mt.GetHighlights()) == 0 {
		mt.selectedMessage = 0
	} else {
		if mt.selectedMessage > 0 {
			mt.selectedMessage--
		}
	}

	mt.Highlight(ms[mt.selectedMessage].ID.String())
	mt.ScrollToHighlight()
}

func (mt *MessagesText) selectFirstAction() {
	ms, err := discordState.Cabinet.Messages(guildsTree.selectedChannel.ID)
	if err != nil {
		log.Println(err)
		return
	}

	mt.selectedMessage = len(ms) - 1
	mt.Highlight(ms[mt.selectedMessage].ID.String())
	mt.ScrollToHighlight()
}

func (mt *MessagesText) selectLastAction() {
	ms, err := discordState.Cabinet.Messages(guildsTree.selectedChannel.ID)
	if err != nil {
		log.Println(err)
		return
	}

	mt.selectedMessage = 0
	mt.Highlight(ms[mt.selectedMessage].ID.String())
	mt.ScrollToHighlight()
}

func (mt *MessagesText) selectReplyAction() {
	if mt.selectedMessage == -1 {
		return
	}

	ms, err := discordState.Cabinet.Messages(guildsTree.selectedChannel.ID)
	if err != nil {
		log.Println(err)
		return
	}

	ref := ms[mt.selectedMessage].ReferencedMessage
	if ref != nil {
		for i, m := range ms {
			if ref.ID == m.ID {
				mt.selectedMessage = i
			}
		}

		mt.Highlight(ms[mt.selectedMessage].ID.String())
		mt.ScrollToHighlight()
	}
}

func (mt *MessagesText) copyContentAction() {
	ms, err := discordState.Cabinet.Messages(guildsTree.selectedChannel.ID)
	if err != nil {
		log.Println(err)
		return
	}

	err = clipboard.WriteAll(ms[mt.selectedMessage].Content)
	if err != nil {
		log.Println(err)
		return
	}
}
