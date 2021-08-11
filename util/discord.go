package util

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

func WriteMessage(v *tview.TextView, clientID discord.UserID, m discord.Message) {
	var b strings.Builder
	// $  ╭ AUTHOR_USERNAME (BOT) MESSAGE_CONTENT*linebreak*
	writeReferencedMessage(&b, clientID, m.ReferencedMessage)
	// $ AUTHOR_USERNAME (BOT)*spacee*
	writeAuthor(&b, clientID, m.Author)
	// $ MESSAGE_CONTENT
	writeContent(&b, m.Content)
	// $ *space*(edited)
	if m.EditedTimestamp.IsValid() {
		b.WriteString(" [::d](edited)[-:-:-]")
	}
	// $ *linebreak*EMBED
	writeEmbeds(&b, m.Embeds)
	// $ *linebreak*ATTACHMENT_URL
	writeAttachments(&b, m.Attachments)

	fmt.Fprintln(v, b.String())
}

func writeEmbeds(b *strings.Builder, embeds []discord.Embed) {
	for range embeds {
		b.WriteString("\n<EMBED>")
	}
}

func writeAttachments(b *strings.Builder, attachments []discord.Attachment) {
	for i := range attachments {
		a := attachments[i]
		b.WriteString("\n")
		b.WriteString("[")
		b.WriteString(a.Filename)
		b.WriteString("]: ")
		b.WriteString(a.URL)
	}
}

func writeAuthor(b *strings.Builder, clientID discord.UserID, u discord.User) {
	if clientID == u.ID {
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

func writeReferencedMessage(b *strings.Builder, clientID discord.UserID, rm *discord.Message) {
	if rm != nil {
		b.WriteRune(' ')
		b.WriteRune('\u256D')
		b.WriteRune(' ')

		if clientID == rm.Author.ID {
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
