package cmd

import (
	"fmt"
	"io"
	"log"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
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
			mt.createBody(mt, *m.ReferencedMessage, true)

			fmt.Fprint(mt, "[::-]\n")
		}

		mt.createHeader(mt, m, false)
		mt.createBody(mt, m, false)
		mt.createFooter(mt, m)

		// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
		fmt.Fprint(mt, `[""]`)
		fmt.Fprintln(mt)
	}
}

func (mt *MessagesText) createHeader(w io.Writer, m discord.Message, isReply bool) {
	time := m.Timestamp.Time().In(time.Local).Format(time.Kitchen)

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

func parseIDsToUsernames(m discord.Message) string {
	var toReplace []string
	for _, mention := range m.Mentions {
		toReplace = append(toReplace,
			fmt.Sprintf("<@%s>", mention.User.ID.String()),
			fmt.Sprintf("__**@%s**__", mention.User.Username),
		)
	}

	return strings.NewReplacer(toReplace...).Replace(m.Content)
}

func (mt *MessagesText) createBody(w io.Writer, m discord.Message, isReply bool) {
	var body string
	if len(m.Mentions) > 0 {
		body = parseIDsToUsernames(m)
	} else {
		body = m.Content
	}

	if isReply {
		fmt.Fprint(w, "[::d]")
	}
	fmt.Fprint(w, markdown.Parse(tview.Escape(body)))
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

func (mt *MessagesText) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch mainFlex.mode {
	case ModeNormal:
		switch event.Name() {
		case cfg.Keys.Normal.MessagesText.Yank:
			if mt.selectedMessage == -1 {
				return nil
			}

			ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
			if err != nil {
				log.Println(err)
				return nil
			}

			err = clipboard.WriteAll(ms[mt.selectedMessage].Content)
			if err != nil {
				log.Println("failed to write to clipboard:", err)
				return nil
			}

			return nil

		case cfg.Keys.Normal.MessagesText.SelectFirst, cfg.Keys.Normal.MessagesText.SelectLast, cfg.Keys.Normal.MessagesText.SelectPrevious, cfg.Keys.Normal.MessagesText.SelectNext, cfg.Keys.Normal.MessagesText.SelectReply:
			ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
			if err != nil {
				log.Println(err)
				return nil
			}

			switch event.Name() {
			case cfg.Keys.Normal.MessagesText.SelectPrevious:
				// If no message is currently selected, select the latest message.
				if len(mt.GetHighlights()) == 0 {
					mt.selectedMessage = 0
				} else {
					if mt.selectedMessage < len(ms)-1 {
						mt.selectedMessage++
					} else {
						return nil
					}
				}
			case cfg.Keys.Normal.MessagesText.SelectNext:
				// If no message is currently selected, select the latest message.
				if len(mt.GetHighlights()) == 0 {
					mt.selectedMessage = 0
				} else {
					if mt.selectedMessage > 0 {
						mt.selectedMessage--
					} else {
						return nil
					}
				}
			case cfg.Keys.Normal.MessagesText.SelectFirst:
				mt.selectedMessage = len(ms) - 1
			case cfg.Keys.Normal.MessagesText.SelectLast:
				mt.selectedMessage = 0
			case cfg.Keys.Normal.MessagesText.SelectReply:
				if mt.selectedMessage == -1 {
					return nil
				}

				if ref := ms[mt.selectedMessage].ReferencedMessage; ref != nil {
					for i, m := range ms {
						if ref.ID == m.ID {
							mt.selectedMessage = i
						}
					}
				}
			}

			mt.Highlight(ms[mt.selectedMessage].ID.String())
			mt.ScrollToHighlight()
			return nil
		case cfg.Keys.Normal.MessagesText.Open:
			if mt.selectedMessage == -1 {
				return nil
			}

			ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
			if err != nil {
				log.Println(err)
				return nil
			}

			attachments := ms[mt.selectedMessage].Attachments
			if len(attachments) == 0 {
				return nil
			}

			for _, a := range attachments {
				go func() {
					if err := open.Start(a.URL); err != nil {
						log.Println(err)
					}
				}()
			}

			return nil
		case cfg.Keys.Normal.MessagesText.Reply, cfg.Keys.Normal.MessagesText.ReplyMention:
			if mt.selectedMessage == -1 {
				return nil
			}

			var title string
			if event.Name() == cfg.Keys.Normal.MessagesText.ReplyMention {
				title += "[@] Replying to "
			} else {
				title += "Replying to "
			}

			ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
			if err != nil {
				log.Println(err)
				return nil
			}

			title += ms[mt.selectedMessage].Author.Tag()
			mainFlex.messageInput.SetTitle(title)
			mainFlex.messageInput.replyMessageIdx = mt.selectedMessage

			app.SetFocus(mainFlex.messageInput)
			return nil
		case cfg.Keys.Normal.MessagesText.Delete:
			if mt.selectedMessage == -1 {
				return nil
			}

			ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
			if err != nil {
				log.Println(err)
				return nil
			}

			m := ms[mt.selectedMessage]
			clientID := discordState.Ready().User.ID

			ps, err := discordState.Permissions(mainFlex.guildsTree.selectedChannelID, discordState.Ready().User.ID)
			if err != nil {
				return nil
			}

			if m.Author.ID != clientID && !ps.Has(discord.PermissionManageMessages) {
				return nil
			}

			if err := discordState.DeleteMessage(mainFlex.guildsTree.selectedChannelID, m.ID, ""); err != nil {
				log.Println(err)
			}

			if err := discordState.MessageRemove(mainFlex.guildsTree.selectedChannelID, m.ID); err != nil {
				log.Println(err)
			}

			ms, err = discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
			if err != nil {
				log.Println(err)
				return nil
			}

			mt.Clear()

			for i := len(ms) - 1; i >= 0; i-- {
				mainFlex.messagesText.createMessage(ms[i])
			}

			return nil
		}

		// do not propagate event to the children in normal mode.
		return nil

	}

	return event
}
