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

	// $ ╭ AUTHOR_USERNAME MESSAGE_CONTENT*linebreak*
	writeReferencedMessage(&b, m.ReferencedMessage)
	// $ AUTHOR_USERNAME (BOT)*space*
	writeAuthor(&b, s, m.Author)

	// $ MESSAGE_CONTENT
	if m.Content != "" {
		b.WriteString(m.Content)
	}

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
		b.WriteString("[#ffb86c::b]")
		b.WriteString(u.Username)
	} else {
		b.WriteString("[#ff5555::b]")
		b.WriteString(u.Username)
	}

	b.WriteString("[-:-:-] ")

	if u.Bot {
		b.WriteString("[#bd93f9]BOT[-:-:-] ")
	}
}

func writeReferencedMessage(b *strings.Builder, rm *discord.Message) {
	if rm != nil {
		b.WriteRune('\u256D')
		b.WriteString(" [#ff5555::d]")
		b.WriteString(rm.Author.Username)
		b.WriteString("[-:-:] ")

		b.WriteString(rm.Content)
		b.WriteString("\n[-:-:-]")
	}
}
