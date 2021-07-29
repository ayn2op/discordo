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
		content.WriteString("[#26BBD9::b]")
		content.WriteString(message.Author.Username)
		content.WriteString("[-:-:-] ")
	} else {
		content.WriteString("[#E95678::b]")
		content.WriteString(message.Author.Username)
		content.WriteString("[-:-:-] ")
	}

	// If the author of the message is a bot account, add "BOT" beside the username of the author.
	if message.Author.Bot {
		content.WriteString("[#29D398]BOT[-:-:-] ")
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

func SendMessage(session *discordgo.Session, channelID string, content string) {
	_, err := session.ChannelMessageSend(channelID, content)
	if err != nil {
		panic(err)
	}
}

func GetMessages(session *discordgo.Session, channelID string, limit int) (messages []*discordgo.Message) {
	messages, err := session.ChannelMessages(channelID, limit, "", "", "")
	if err != nil {
		panic(err)
	}

	return
}
