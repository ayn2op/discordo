package util

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/rivo/tview"
)

func WriteMessage(v *tview.TextView, s *state.State, m discord.Message) {
	var b strings.Builder

	if s.Ready().User.ID == m.Author.ID {
		b.WriteString("[#ffb86c::b]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-:-:-] ")
	} else {
		b.WriteString("[#ff5555::b]")
		b.WriteString(m.Author.Username)
		b.WriteString("[-:-:-] ")
	}

	if m.Author.Bot {
		b.WriteString("[#bd93f9]BOT[-:-:-] ")
	}

	if m.Content != "" {
		b.WriteString(m.Content)
	}

	if m.EditedTimestamp.IsValid() {
		b.WriteString(" [::d](edited)[-:-:-]")
	}

	// TODO(rigormorrtiss): display the message embed using "special" format
	if len(m.Embeds) > 0 {
		b.WriteString("\n<EMBED(S)>")
	}

	attachments := m.Attachments
	for i := range attachments {
		b.WriteString("\n")
		b.WriteString(attachments[i].URL)
	}

	fmt.Fprintln(v, b.String())
}
