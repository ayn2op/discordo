package util

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/rivo/tview"
)

func WriteMessage(messagesTextView *tview.TextView, state *state.State, message discord.Message) {
	var parsedMsg strings.Builder

	if state.Ready().User.ID == message.Author.ID {
		parsedMsg.WriteString("[#ffb86c::b]")
		parsedMsg.WriteString(message.Author.Username)
		parsedMsg.WriteString("[-:-:-] ")
	} else {
		parsedMsg.WriteString("[#ff5555::b]")
		parsedMsg.WriteString(message.Author.Username)
		parsedMsg.WriteString("[-:-:-] ")
	}

	if message.Author.Bot {
		parsedMsg.WriteString("[#bd93f9]BOT[-:-:-] ")
	}

	if message.Content != "" {
		parsedMsg.WriteString(message.Content)
	}

	if message.EditedTimestamp.IsValid() {
		parsedMsg.WriteString(" [::d](edited)[-:-:-]")
	}

	// TODO(rigormorrtiss): display the message embed using "special" format
	if len(message.Embeds) > 0 {
		parsedMsg.WriteString("\n<EMBED(S)>")
	}

	attachments := message.Attachments
	for i := range attachments {
		parsedMsg.WriteString("\n")
		parsedMsg.WriteString(attachments[i].URL)
	}

	fmt.Fprintln(messagesTextView, parsedMsg.String())
}
