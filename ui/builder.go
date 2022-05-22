package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/ayntgl/astatine"
	"github.com/ayntgl/discordo/discord"
)

func buildMessage(app *App, m *astatine.Message) []byte {
	var b strings.Builder

	switch m.Type {
	case astatine.MessageTypeDefault, astatine.MessageTypeReply:
		// Define a new region and assign message ID as the region ID.
		// Learn more:
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		b.WriteString("[\"")
		b.WriteString(m.ID)
		b.WriteString("\"]")
		// Build the message associated with crosspost, channel follow add, pin, or a reply.
		buildReferencedMessage(&b, m.ReferencedMessage, app.Session.State.User.ID)

		if app.Config.Timestamps {
			b.WriteString("[::d]")
			b.WriteString(m.Timestamp.Format(time.Stamp))
			b.WriteString("[::-]")
			b.WriteByte(' ')
		}

		// Build the author of this message.
		buildAuthor(&b, m.Author, app.Session.State.User.ID)

		// Build the contents of the message.
		buildContent(&b, m, app.Session.State.User.ID)

		if m.EditedTimestamp != nil {
			b.WriteString(" [::d](edited)[::-]")
		}

		// Build the embeds associated with the message.
		buildEmbeds(&b, m.Embeds)

		// Build the message attachments (attached files to the message).
		buildAttachments(&b, m.Attachments)

		// Tags with no region ID ([""]) do not start new regions. They can
		// therefore be used to mark the end of a region.
		b.WriteString("[\"\"]")

		b.WriteByte('\n')
	case astatine.MessageTypeGuildMemberJoin:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-] joined the server.")

		b.WriteByte('\n')
	case astatine.MessageTypeCall:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-] started a call.")

		b.WriteByte('\n')
	case astatine.MessageTypeChannelPinnedMessage:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-] pinned a message.")

		b.WriteByte('\n')
	}

	if str := b.String(); str != "" {
		b := make([]byte, len(str)+1)
		copy(b, str)

		return b
	}

	return nil
}

func buildReferencedMessage(b *strings.Builder, rm *astatine.Message, clientID string) {
	if rm != nil {
		b.WriteString(" ╭ ")
		b.WriteString("[::d]")
		buildAuthor(b, rm.Author, clientID)

		if rm.Content != "" {
			rm.Content = buildMentions(rm.Content, rm.Mentions, clientID)
			b.WriteString(discord.ParseMarkdown(rm.Content))
		}

		b.WriteString("[::-]")
		b.WriteByte('\n')
	}
}

func buildContent(b *strings.Builder, m *astatine.Message, clientID string) {
	if m.Content != "" {
		m.Content = buildMentions(m.Content, m.Mentions, clientID)
		b.WriteString(discord.ParseMarkdown(m.Content))
	}
}

func buildEmbeds(b *strings.Builder, es []*astatine.MessageEmbed) {
	for _, e := range es {
		if e.Type != astatine.EmbedTypeRich {
			continue
		}

		var (
			embedBuilder strings.Builder
			hasHeading   bool
		)
		prefix := fmt.Sprintf("[#%06X]▐[-] ", e.Color)

		b.WriteByte('\n')
		embedBuilder.WriteString(prefix)

		if e.Author != nil {
			hasHeading = true
			embedBuilder.WriteString("[::u]")
			embedBuilder.WriteString(e.Author.Name)
			embedBuilder.WriteString("[::-]")
		}

		if e.Title != "" {
			if hasHeading {
				embedBuilder.WriteByte('\n')
				embedBuilder.WriteByte('\n')
			}

			embedBuilder.WriteString("[::b]")
			embedBuilder.WriteString(e.Title)
			embedBuilder.WriteString("[::-]")
		}

		if e.Description != "" {
			if hasHeading {
				embedBuilder.WriteByte('\n')
				embedBuilder.WriteByte('\n')
			}

			embedBuilder.WriteString(discord.ParseMarkdown(e.Description))
		}

		if len(e.Fields) != 0 {
			if hasHeading || e.Description != "" {
				embedBuilder.WriteByte('\n')
				embedBuilder.WriteByte('\n')
			}

			for i, ef := range e.Fields {
				embedBuilder.WriteString("[::b]")
				embedBuilder.WriteString(ef.Name)
				embedBuilder.WriteString("[::-]")
				embedBuilder.WriteByte('\n')
				embedBuilder.WriteString(discord.ParseMarkdown(ef.Value))

				if i != len(e.Fields)-1 {
					embedBuilder.WriteString("\n\n")
				}
			}
		}

		if e.Footer != nil {
			if hasHeading {
				embedBuilder.WriteString("\n\n")
			}

			embedBuilder.WriteString(e.Footer.Text)
		}

		b.WriteString(strings.ReplaceAll(embedBuilder.String(), "\n", "\n"+prefix))
	}
}

func buildAttachments(b *strings.Builder, as []*astatine.MessageAttachment) {
	for _, a := range as {
		b.WriteByte('\n')
		b.WriteByte('[')
		b.WriteString(a.Filename)
		b.WriteString("]: ")
		b.WriteString(a.URL)
	}
}

func buildMentions(content string, mentions []*astatine.User, clientID string) string {
	for _, mUser := range mentions {
		var color string
		if mUser.ID == clientID {
			color = "[:#5865F2]"
		} else {
			color = "[#EB459E]"
		}

		content = strings.NewReplacer(
			// <@USER_ID>
			"<@"+mUser.ID+">",
			color+"@"+mUser.Username+"[-:-]",
			// <@!USER_ID>
			"<@!"+mUser.ID+">",
			color+"@"+mUser.Username+"[-:-]",
		).Replace(content)
	}

	return content
}

func buildAuthor(b *strings.Builder, u *astatine.User, clientID string) {
	if u.ID == clientID {
		b.WriteString("[#57F287]")
	} else {
		b.WriteString("[#ED4245]")
	}

	b.WriteString(u.Username)
	b.WriteString("[-] ")
	// If the message author is a bot account, render the message with bot label
	// for distinction.
	if u.Bot {
		b.WriteString("[#EB459E]BOT[-] ")
	}
}
