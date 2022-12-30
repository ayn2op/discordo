package main

import (
	"fmt"
	"strings"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

type MessagesText struct {
	*tview.TextView

	builder strings.Builder
}

func newMessagesText() *MessagesText {
	mt := &MessagesText{
		TextView: tview.NewTextView(),
	}

	mt.SetDynamicColors(true)
	mt.SetRegions(true)

	mt.SetBorder(true)
	mt.SetBorderPadding(cfg.BorderPadding())

	return mt
}

func (mt *MessagesText) newMessage(m *discord.Message) {
	switch m.Type {
	case discord.DefaultMessage:
		{
			mt.builder.WriteString("[blue::b]")
			mt.builder.WriteString(m.Author.Username)
			mt.builder.WriteString("[-:-:-]")
		}

		mt.builder.WriteByte('\n')
		mt.builder.WriteByte('\n')
	}

	fmt.Fprintln(mt, mt.builder.String())
}
