package cmd

import (
	"fmt"
	"io"
	"log"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/markdown"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
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

func (mt *MessagesText) drawMsgs(cID discord.ChannelID) {
	ms, err := discordState.Messages(cID, uint(cfg.MessagesLimit))
	if err != nil {
		log.Println(err)
		return
	}

	for i := len(ms) - 1; i >= 0; i-- {
		mainFlex.messagesText.createMessage(ms[i])
	}
}

func (mt *MessagesText) reset() {
	mainFlex.messagesText.selectedMessage = -1

	mt.SetTitle("")
	mt.Clear()
	mt.Highlight()
}

func (mt *MessagesText) createMessage(m discord.Message) {
	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		// Region tags are square brackets that contain a region ID in double quotes
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		fmt.Fprintf(mt, `["%s"]`, m.ID)

		if m.ReferencedMessage != nil {
			mt.createHeader(mt, *m.ReferencedMessage, true)
			mt.createBody(mt, *m.ReferencedMessage)

			fmt.Fprint(mt, "[::-]\n")
		}

		mt.createHeader(mt, m, false)
		mt.createBody(mt, m)
		mt.createFooter(mt, m)

		// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
		fmt.Fprint(mt, `[""]`)
		fmt.Fprintln(mt)
	}
}

func (mt *MessagesText) createHeader(w io.Writer, m discord.Message, isReply bool) {
	time := m.Timestamp.Format(time.Kitchen)

	if cfg.Timestamps && cfg.TimestampsBeforeAuthor {
		fmt.Fprintf(w, "[::d]%7s[::-] ", time)
	}

	if isReply {
		fmt.Fprintf(mt, "[::d]%s", cfg.Theme.MessagesText.ReplyIndicator)
	}

	fmt.Fprintf(w, "[%s]%s[-:-:-] ", cfg.Theme.MessagesText.AuthorColor, m.Author.Username)

	if cfg.Timestamps && !cfg.TimestampsBeforeAuthor {
		fmt.Fprintf(w, "[::d]%s[::-] ", time)
	}
}

func (mt *MessagesText) createBody(w io.Writer, m discord.Message) {
	fmt.Fprint(w, markdown.Parse(tview.Escape(m.Content)))
}

func (mt *MessagesText) createFooter(w io.Writer, m discord.Message) {
	for _, a := range m.Attachments {
		fmt.Fprintln(w)
		fmt.Fprintf(w, "[%s]: %s", a.Filename, a.URL)
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
	case cfg.Keys.MessagesText.ShowImage:
		mt.showImageAction()
		return nil
	case cfg.Keys.MessagesText.Delete:
		mt.deleteAction()
		return nil
	case cfg.Keys.Cancel:
		mainFlex.guildsTree.selectedChannelID = 0

		mainFlex.messagesText.reset()
		mainFlex.messageInput.reset()
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

	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		log.Println(err)
		return
	}

	title += ms[mt.selectedMessage].Author.Tag()
	mainFlex.messageInput.SetTitle(title)

	app.SetFocus(mainFlex.messageInput)
}

func (mt *MessagesText) selectPreviousAction() {
	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
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
		} else {
			return
		}
	}

	mt.Highlight(ms[mt.selectedMessage].ID.String())
	mt.ScrollToHighlight()
}

func (mt *MessagesText) selectNextAction() {
	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
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
		} else {
			return
		}
	}

	mt.Highlight(ms[mt.selectedMessage].ID.String())
	mt.ScrollToHighlight()
}

func (mt *MessagesText) selectFirstAction() {
	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		log.Println(err)
		return
	}

	mt.selectedMessage = len(ms) - 1
	mt.Highlight(ms[mt.selectedMessage].ID.String())
	mt.ScrollToHighlight()
}

func (mt *MessagesText) selectLastAction() {
	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
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

	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
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
	if mt.selectedMessage == -1 {
		return
	}

	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
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

func (mt *MessagesText) showImageAction() {
	if mt.selectedMessage == -1 {
		return
	}

	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		log.Println(err)
		return
	}

	as := ms[mt.selectedMessage].Attachments
	if len(as) == 0 {
		return
	}

	ai, err := newAttachmentImage(as[0])
	if err != nil {
		log.Println(err)
		return
	}

	app.SetRoot(ai, true)
}

func (mt *MessagesText) deleteAction() {
	if mt.selectedMessage == -1 {
		return
	}

	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		log.Println(err)
		return
	}

	m := ms[mt.selectedMessage]
	if err := discordState.DeleteMessage(mainFlex.guildsTree.selectedChannelID, m.ID, ""); err != nil {
		log.Println(err)
	}

	if err := discordState.MessageRemove(mainFlex.guildsTree.selectedChannelID, m.ID); err != nil {
		log.Println(err)
	}

	ms, err = discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		log.Println(err)
		return
	}

	mt.Clear()

	for i := len(ms) - 1; i >= 0; i-- {
		mainFlex.messagesText.createMessage(ms[i])
	}
}
