package util

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/diamondburned/arikawa/v2/discord"
	"github.com/diamondburned/arikawa/v2/session"
	"github.com/rivo/tview"
)

func WriteMessage(messagesTextView *tview.TextView, message discord.Message) {
	var content strings.Builder

	content.WriteString("[red::b]" + message.Author.Username + "[-:-:-] ")
	// If the author of the message is a bot account, add "BOT" beside the username of the author.
	if message.Author.Bot {
		content.WriteString("[blue]BOT[-:-:-] ")
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
		content.WriteString("\n" + attachments[i].URL)
	}

	fmt.Fprintln(messagesTextView, content.String())
}

func SendMessage(session *session.Session, channelID discord.ChannelID, content string) {
	_, err := session.SendText(channelID, content)
	if err != nil {
		panic(err)
	}
}

func GetMessages(session *session.Session, channelID discord.ChannelID, limit uint) (messages []discord.Message) {
	messages, err := session.Messages(channelID, limit)
	if err != nil {
		panic(err)
	}

	return
}
