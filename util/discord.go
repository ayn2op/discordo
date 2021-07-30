package util

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/bwmarrin/discordgo"
	"github.com/rivo/tview"
)

func WriteMessage(messagesTextView *tview.TextView, session *discordgo.Session, message *discordgo.Message) {
	var content strings.Builder

	if session.State.User.ID == message.Author.ID {
		content.WriteString("[#ffb86c::b]")
		content.WriteString(message.Author.Username)
		content.WriteString("[-:-:-] ")
	} else {
		content.WriteString("[#ff5555::b]")
		content.WriteString(message.Author.Username)
		content.WriteString("[-:-:-] ")
	}

	// If the author of the message is a bot account, add "BOT" beside the username of the author.
	if message.Author.Bot {
		content.WriteString("[#bd93f9]BOT[-:-:-] ")
	}

	if message.Content != "" {
		content.WriteString(message.Content)
	}

	// TODO(rigormorrtiss): display the message embed using "special" format
	if len(message.Embeds) > 0 {
		content.WriteString("\n<EMBED>")
	}

	attachments := message.Attachments
	for i := range attachments {
		content.WriteString("\n")
		content.WriteString(attachments[i].URL)
	}

	fmt.Fprintln(messagesTextView, content.String())
}
