package util

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

func WriteMessage(v *tview.TextView, clientID discord.UserID, m discord.Message) {
	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		var b strings.Builder
		// $  ╭ AUTHOR_USERNAME (BOT) MESSAGE_CONTENT*linebreak*
		writeReferencedMessage(&b, clientID, m.ReferencedMessage)
		// $ AUTHOR_USERNAME (BOT)*spacee*
		writeAuthor(&b, clientID, m.Author)
		// $ MESSAGE_CONTENT
		if m.Content != "" {
			m.Content = parseMessageMentions(m.Content, m.Mentions, clientID)
			b.WriteString(m.Content)
		}
		// $ *space*(edited)
		if m.EditedTimestamp.IsValid() {
			b.WriteString(" [::d](edited)[::-]")
		}
		// $ *linebreak*EMBED
		writeEmbeds(&b, m.Embeds)
		// $ *linebreak*ATTACHMENT_URL
		writeAttachments(&b, m.Attachments)

		fmt.Fprintln(v, b.String())
	case discord.ThreadStarterMessage:
		WriteMessage(v, clientID, *m.ReferencedMessage)
	}
}

func parseMessageMentions(content string, mentions []discord.GuildUser, clientID discord.UserID) string {
	for _, mUser := range mentions {
		var color string
		if mUser.ID == clientID {
			color = "[#000000:#FEE75C]"
		} else {
			color = "[:#5865F2]"
		}

		content = strings.NewReplacer(
			// <@USER_ID>
			"<@"+mUser.ID.String()+">",
			color+"@"+mUser.Username+"[-:-]",
			// <@!USER_ID>
			"<@!"+mUser.ID.String()+">",
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
	for _, a := range attachments {
		b.WriteString("\n[")
		b.WriteString(a.Filename)
		b.WriteString("]: ")
		b.WriteString(a.URL)
	}
}

func writeAuthor(b *strings.Builder, clientID discord.UserID, u discord.User) {
	if u.ID == clientID {
		b.WriteString("[#57F287]")
	} else {
		b.WriteString("[#ED4245]")
	}

	b.WriteString(u.Username)
	b.WriteString("[-] ")

	if u.Bot {
		b.WriteString("[#EB459E]BOT[-] ")
	}
}

func writeReferencedMessage(b *strings.Builder, clientID discord.UserID, rm *discord.Message) {
	if rm != nil {
		b.WriteString(" ╭ ")

		if rm.Author.ID == clientID {
			b.WriteString("[#57F287::d]")
		} else {
			b.WriteString("[#ED4245::d]")
		}

		b.WriteString(rm.Author.Username)
		b.WriteString("[-] ")

		if rm.Author.Bot {
			b.WriteString("[#EB459E]BOT[-] ")
		}

		if rm.Content != "" {
			rm.Content = parseMessageMentions(rm.Content, rm.Mentions, clientID)
			b.WriteString(rm.Content)
		}
		b.WriteString("[::-]\n")
	}
}
