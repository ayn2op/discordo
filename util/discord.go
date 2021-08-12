package util

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

func WriteMessage(v *tview.TextView, clientID discord.UserID, m discord.Message) {
	m.Content = parseMessageMentions(m.Content, m.Mentions, clientID)

	var b strings.Builder
	// $  ╭ AUTHOR_USERNAME (BOT) MESSAGE_CONTENT*linebreak*
	writeReferencedMessage(&b, clientID, m.ReferencedMessage)
	// $ AUTHOR_USERNAME (BOT)*spacee*
	writeAuthor(&b, clientID, m.Author)
	// $ MESSAGE_CONTENT
	b.WriteString(m.Content)
	// $ *space*(edited)
	if m.EditedTimestamp.IsValid() {
		b.WriteString(" [::d](edited)[::-]")
	}
	// $ *linebreak*EMBED
	writeEmbeds(&b, m.Embeds)
	// $ *linebreak*ATTACHMENT_URL
	writeAttachments(&b, m.Attachments)

	fmt.Fprintln(v, b.String())
}

func parseMessageMentions(content string, mentions []discord.GuildUser, clientID discord.UserID) string {
	for i := range mentions {
		mUser := mentions[i]

		var color string
		if mUser.ID == clientID {
			color = "[#000000:#FEE75C]"
		} else {
			color = "[:#5865F2]"
		}

		content = strings.NewReplacer(
			// <@!USER_ID>
			fmt.Sprintf("<@!%d>", mUser.ID),
			color+"@"+mUser.Username+"[-:-]",
			// <@USER_ID>
			fmt.Sprintf("<@%d>", mUser.ID),
			color+"@"+mUser.Username+"[-:-]",
		).Replace(content)
	}

	return content
}

func writeEmbeds(b *strings.Builder, embeds []discord.Embed) {
	for range embeds {
		b.WriteString("\n<EMBED>")
	}
}

func writeAttachments(b *strings.Builder, attachments []discord.Attachment) {
	for i := range attachments {
		a := attachments[i]
		b.WriteString("\n[" + a.Filename + "]: ")
		b.WriteString(a.URL)
	}
}

func writeAuthor(b *strings.Builder, clientID discord.UserID, u discord.User) {
	if u.ID == clientID {
		b.WriteString("[#57F287]")
	} else {
		b.WriteString("[#ED4245]")
	}

	b.WriteString(u.Username + "[-] ")

	if u.Bot {
		b.WriteString("[#EB459E]BOT[-] ")
	}
}

func writeReferencedMessage(b *strings.Builder, clientID discord.UserID, rm *discord.Message) {
	if rm != nil {
		rm.Content = parseMessageMentions(rm.Content, rm.Mentions, clientID)

		b.WriteRune(' ')
		b.WriteRune('\u256D')
		b.WriteRune(' ')

		if rm.Author.ID == clientID {
			b.WriteString("[#57F287::d]")
		} else {
			b.WriteString("[#ED4245::d]")
		}

		b.WriteString(rm.Author.Username + "[-] ")

		if rm.Author.Bot {
			b.WriteString("[#EB459E]BOT[-] ")
		}
		// Reset foreground
		b.WriteString(rm.Content + "[::-]\n")
	}
}
