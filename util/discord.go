package util

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

func WriteMessage(v *tview.TextView, clientID discord.UserID, m discord.Message) {
	var b strings.Builder
	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		parseMessage(v, &b, m, clientID)
		fmt.Fprintln(v, b.String())
	case discord.ThreadStarterMessage:
		parseMessage(v, &b, *m.ReferencedMessage, clientID)
		fmt.Fprintln(v, b.String())
	case discord.GuildMemberJoinMessage:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-]")
		b.WriteString(" joined the server")
		fmt.Fprintln(v, b.String())
	}
}

func parseMessage(v *tview.TextView, b *strings.Builder, m discord.Message, clientID discord.UserID) {
	// $  ╭ AUTHOR_USERNAME (BOT) MESSAGE_CONTENT*linebreak*
	parseReferencedMessage(b, clientID, m.ReferencedMessage)
	// $ AUTHOR_USERNAME (BOT)*spacee*
	parseAuthor(b, clientID, m.Author)
	// $ MESSAGE_CONTENT
	parseContent(b, m, clientID)
	// $ *space*(edited)
	parseEditedTimestamp(b, m.EditedTimestamp)
	// $ *linebreak*EMBED
	parseEmbeds(b, m.Embeds)
	// $ *linebreak*ATTACHMENT_URL
	parseAttachments(b, m.Attachments)
}

func parseContent(b *strings.Builder, m discord.Message, clientID discord.UserID) {
	if m.Content != "" {
		m.Content = parseMessageMentions(m.Content, m.Mentions, clientID)
		b.WriteString(m.Content)
	}
}

func parseEditedTimestamp(b *strings.Builder, t discord.Timestamp) {
	if t.IsValid() {
		b.WriteString(" [::d](edited)[::-]")
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

func parseEmbeds(b *strings.Builder, embeds []discord.Embed) {
	for range embeds {
		b.WriteString("\n<EMBED>")
	}
}

func parseAttachments(b *strings.Builder, attachments []discord.Attachment) {
	for _, a := range attachments {
		b.WriteString("\n[")
		b.WriteString(a.Filename)
		b.WriteString("]: ")
		b.WriteString(a.URL)
	}
}

func parseAuthor(b *strings.Builder, clientID discord.UserID, u discord.User) {
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

func parseReferencedMessage(b *strings.Builder, clientID discord.UserID, rm *discord.Message) {
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
