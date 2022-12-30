package main

import (
	"fmt"
	"sync"
	"text/template"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
)

var p = sync.Pool{
	New: func() any {
		return new(template.Template)
	},
}

type MessagesText struct {
	*tview.TextView
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

func (mt *MessagesText) newMessage(m *discord.Message) error {
	switch m.Type {
	case discord.DefaultMessage:
		// Region tags are square brackets that contain a region ID in double quotes
		// https://pkg.go.dev/github.com/rivo/tview#hdr-Regions_and_Highlights
		fmt.Fprintf(mt, `["%s"]`, m.ID)

		if m.ReferencedMessage != nil {
			fmt.Fprintf(mt, "[blue::bd]%s[-:-:-] [::-]", m.ReferencedMessage.Author.Username)
			fmt.Fprint(mt, m.ReferencedMessage.Content)
			fmt.Fprintln(mt)
		}

		fmt.Fprintf(mt, "[blue::b]%s[-:-:-] ", m.Author.Username)
		fmt.Fprint(mt, m.Content)
		// Tags with no region ID ([""]) don't start new regions. They can therefore be used to mark the end of a region.
		fmt.Fprint(mt, `[""]`)

		fmt.Fprintln(mt)
	}

	return nil
}
