package ui

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/diamondburned/arikawa/v3/discord"
)

func buildMessage(app *App, m discord.Message) {
	switch m.Type {
	case discord.DefaultMessage, discord.InlinedReplyMessage:
		loc, err := time.LoadLocation(app.Config.Timezone)
		if err != nil {
			return
		}

		app.Config.Template.Execute(app.MessagesPanel, struct {
			Timestamp string
			Content   string
			UserID    discord.UserID
			Message   discord.Message
		}{
			Timestamp: m.Timestamp.Time().In(loc).Format(app.Config.TimeFormat),
			Content:   parseMarkdown(buildMentions(m.Content, m.Mentions, app.State.Ready().User.ID)),
			UserID:    app.State.Ready().User.ID,
			Message:   m,
		})

		// Build the embeds associated with the message.
		buildEmbeds(app.MessagesPanel, m.Embeds)
		// Build the message attachments (attached files to the message).
		buildAttachments(app.MessagesPanel, m.Attachments)
	case discord.GuildMemberJoinMessage:
		fmt.Fprintf(app.MessagesPanel, "[#5865F2]%s[-] joined the server.\n\n", m.Author.Username)
	case discord.CallMessage:
		fmt.Fprintf(app.MessagesPanel, "[#5865F2]%s[-] started a call.\n\n", m.Author.Username)
	case discord.ChannelPinnedMessage:
		fmt.Fprintf(app.MessagesPanel, "[#5865F2]%s[-] pinned a message.\n\n", m.Author.Username)
	}
}

func buildEmbeds(w io.Writer, es []discord.Embed) {
	for _, e := range es {
		if e.Type != discord.NormalEmbed {
			continue
		}

		var (
			embedBuilder strings.Builder
			hasHeading   bool
		)
		prefix := fmt.Sprintf("[#%06X]▐[-] ", e.Color)

		fmt.Fprintln(w)
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

			embedBuilder.WriteString(parseMarkdown(e.Description))
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
				embedBuilder.WriteString(parseMarkdown(ef.Value))

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

		fmt.Fprint(w, strings.ReplaceAll(embedBuilder.String(), "\n", "\n"+prefix))
	}
}

func buildAttachments(w io.Writer, as []discord.Attachment) {
	for _, a := range as {
		fmt.Fprintf(w, "\n[%s]: %s\n\n", a.Filename, a.URL)
	}
}

func buildMentions(content string, mentions []discord.GuildUser, clientID discord.UserID) string {
	for _, mUser := range mentions {
		var color string
		if mUser.ID == clientID {
			color = "[:#5865F2]"
		} else {
			color = "[#EB459E]"
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
