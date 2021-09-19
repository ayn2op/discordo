package main

import (
	"fmt"
	"strings"

	"github.com/bwmarrin/discordgo"
)

func renderMessages(cID string) {
	ms, err := session.ChannelMessages(cID, conf.GetMessagesLimit, "", "", "")
	if err != nil {
		return
	}

	for i := len(ms) - 1; i >= 0; i-- {
		selectedChannel.Messages = append(selectedChannel.Messages, ms[i])
		renderMessage(ms[i])
	}
}

func renderMessage(m *discordgo.Message) {
	var b strings.Builder

	switch m.Type {
	case discordgo.MessageTypeDefault, discordgo.MessageTypeReply:
		// Define a new region and assign message ID as the region ID.
		// Learn more:
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		b.WriteString("[\"")
		b.WriteString(m.ID)
		b.WriteString("\"]")
		// Render the message associated with crosspost, channel follow add,
		// pin, or a reply.
		if rm := m.ReferencedMessage; rm != nil {
			b.WriteString(" ╭ ")
			b.WriteString("[::d]")
			parseAuthor(&b, rm.Author)

			if rm.Content != "" {
				rm.Content = parseMentions(rm.Content, rm.Mentions)
				b.WriteString(rm.Content)
			}

			b.WriteString("[::-]\n")
		}
		// Render the author of the message.
		parseAuthor(&b, m.Author)
		// If the message content is not empty, parse the message mentions
		// (users mentioned in the message) and render the message content.
		if m.Content != "" {
			m.Content = parseMentions(m.Content, m.Mentions)
			b.WriteString(m.Content)
		}
		// If the edited timestamp of the message is not empty; it implies that
		// the message has been edited, hence render the message with edited
		// label for distinction
		if m.EditedTimestamp != "" {
			b.WriteString(" [::d](edited)[::-]")
		}
		// TODO: render message embeds
		for range m.Embeds {
			b.WriteString("\n<EMBED>")
		}
		// Render the message attachments (attached files to the message).
		for _, a := range m.Attachments {
			b.WriteString("\n[")
			b.WriteString(a.Filename)
			b.WriteString("]: ")
			b.WriteString(a.URL)
		}
		// Tags with no region ID ([""]) do not start new regions. They can
		// therefore be used to mark the end of a region.
		b.WriteString("[\"\"]")

		fmt.Fprintln(messagesTextView, b.String())
	case discordgo.MessageTypeGuildMemberJoin:
		b.WriteString("[#5865F2]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-] joined the server")

		fmt.Fprintln(messagesTextView, b.String())
	}
}

func parseMentions(content string, mentions []*discordgo.User) string {
	for _, mUser := range mentions {
		var color string
		if mUser.ID == session.State.User.ID {
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

func parseAuthor(b *strings.Builder, u *discordgo.User) {
	if u.ID == session.State.User.ID {
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
