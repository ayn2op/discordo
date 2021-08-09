package util

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/rivo/tview"
)

func WriteMessage(v *tview.TextView, s *state.State, m discord.Message) {
	var b strings.Builder
	// $  ╭ AUTHOR_USERNAME (BOT) MESSAGE_CONTENT*linebreak*
	writeReferencedMessage(&b, s, m.ReferencedMessage)
	// $ AUTHOR_USERNAME (BOT)*spacee*
	writeAuthor(&b, s, m.Author)
	// $ MESSAGE_CONTENT
	writeContent(&b, m.Content)
	// $ *space*(edited)
	if m.EditedTimestamp.IsValid() {
		b.WriteString(" [::d](edited)[-:-:-]")
	}
	// TODO(rigormorrtiss): display the message embed using "special" format
	if len(m.Embeds) > 0 {
		b.WriteString("\n<EMBED(S)>")
	}
	// $ *linebreak*ATTACHMENT_URL
	writeAttachments(&b, m.Attachments)

	fmt.Fprintln(v, b.String())
}

func writeAttachments(b *strings.Builder, attachments []discord.Attachment) {
	for i := range attachments {
		b.WriteString("\n")
		b.WriteString(attachments[i].URL)
	}
}

func writeAuthor(b *strings.Builder, s *state.State, u discord.User) {
	if s.Ready().User.ID == u.ID {
		b.WriteString("[#59E3E3]")
	} else {
		b.WriteString("[#E95678]")
	}

	b.WriteString(u.Username)
	b.WriteString("[-:-:-] ")

	if u.Bot {
		b.WriteString("[#59E3E3]BOT[-:-:-] ")
	}
}

func writeReferencedMessage(b *strings.Builder, s *state.State, rm *discord.Message) {
	if rm != nil {
		b.WriteRune(' ')
		b.WriteRune('\u256D')
		b.WriteRune(' ')

		if s.Ready().User.ID == rm.Author.ID {
			b.WriteString("[#59E3E3::d]")
		} else {
			b.WriteString("[#E95678::d]")
		}

		b.WriteString(rm.Author.Username)
		// Reset foreground
		b.WriteString("[-::] ")

		writeContent(b, rm.Content)
		b.WriteString("[-:-:-]\n")
	}
}

func writeContent(b *strings.Builder, c string) {
	if c != "" {
		c = tview.Escape(c)
		b.WriteString(c)
	}
}
