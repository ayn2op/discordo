package main

import (
	"log"
	"strings"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessageInput struct {
	*tview.InputField
}

func newMessageInput() *MessageInput {
	mi := &MessageInput{
		InputField: tview.NewInputField(),
	}

	mi.SetInputCapture(mi.onInputCapture)
	mi.SetFieldBackgroundColor(tcell.GetColor(cfg.Theme.MessageInput.BackgroundColor))
	mi.SetBackgroundColor(tcell.GetColor(cfg.Theme.MessageInput.BackgroundColor))

	mi.SetTitleColor(tcell.GetColor(cfg.Theme.MessageInput.TitleColor))
	mi.SetTitleAlign(tview.AlignLeft)

	padding := cfg.Theme.MessageInput.BorderPadding
	mi.SetBorder(cfg.Theme.MessageInput.Border)
	mi.SetBorderPadding(padding[0], padding[1], padding[2], padding[3])

	return mi
}

func (mi *MessageInput) reset() {
	mi.SetTitle("")
	mi.SetText("")
}

func (mi *MessageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case cfg.Keys.MessageInput.Send:
		mi.sendAction()
		return nil
	case cfg.Keys.MessageInput.Cancel:
		mi.reset()
		return nil
	}

	return event
}

func (mi *MessageInput) sendAction() {
	if guildsTree.selectedChannel == nil {
		return
	}

	text := strings.TrimSpace(mi.GetText())
	if text == "" {
		return
	}

	var err error

	if messagesText.selectedMessage != nil {
		title := mi.GetTitle()
		if title == "" {
			return
		}

		data := api.SendMessageData{
			Content:         text,
			Reference:       &discord.MessageReference{MessageID: messagesText.selectedMessage.ID},
			AllowedMentions: &api.AllowedMentions{RepliedUser: option.False},
		}

		if strings.HasPrefix(title, "[@]") {
			data.AllowedMentions.RepliedUser = option.True
		}

		_, err = discordState.SendMessageComplex(guildsTree.selectedChannel.ID, data)
	} else {
		_, err = discordState.SendMessage(guildsTree.selectedChannel.ID, text)
	}

	if err != nil {
		log.Println(err)
		return
	}

	messageInput.reset()
}
