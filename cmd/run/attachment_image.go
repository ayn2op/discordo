package run

import (
	"image"
	_ "image/jpeg"
	_ "image/png"
	"net/http"

	"github.com/ayn2op/discordo/config"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type AttachmentImage struct {
	*tview.Image
}

func newAttachmentImage(a discord.Attachment) (*AttachmentImage, error) {
	ai := &AttachmentImage{
		Image: tview.NewImage(),
	}

	ai.SetInputCapture(ai.onInputCapture)
	ai.SetBackgroundColor(tcell.GetColor(config.Current.Theme.BackgroundColor))
	ai.SetTitleColor(tcell.GetColor(config.Current.Theme.TitleColor))
	ai.SetTitleAlign(tview.AlignLeft)

	p := config.Current.Theme.BorderPadding
	ai.SetBorder(config.Current.Theme.Border)
	ai.SetBorderColor(tcell.GetColor(config.Current.Theme.BorderColor))
	ai.SetBorderPadding(p[0], p[1], p[2], p[3])

	resp, err := http.Get(a.URL)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	i, _, err := image.Decode(resp.Body)
	if err != nil {
		return nil, err
	}

	ai.SetTitle(a.Filename)
	ai.SetImage(i)
	return ai, nil
}

func (ai *AttachmentImage) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	if event.Name() == config.Current.Keys.Cancel {
		app.SetRoot(mainFlex, true)
		app.SetFocus(mainFlex.messagesText)
		return nil
	}

	return event
}
