package main

import (
	"log"
	"os"
	"os/exec"
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
	mi.SetFieldBackgroundColor(tcell.GetColor(config.Theme.MessageInput.BackgroundColor))
	mi.SetBackgroundColor(tcell.GetColor(config.Theme.MessageInput.BackgroundColor))

	mi.SetTitleColor(tcell.GetColor(config.Theme.MessageInput.TitleColor))
	mi.SetTitleAlign(tview.AlignLeft)

	padding := config.Theme.MessageInput.BorderPadding
	mi.SetBorder(config.Theme.MessageInput.Border)
	mi.SetBorderPadding(padding[0], padding[1], padding[2], padding[3])

	return mi
}

func (mi *MessageInput) reset() {
	mi.SetTitle("")
	mi.SetText("")
}

func (mi *MessageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case config.Keys.MessageInput.Send:
		mi.sendAction()
		return nil
	case config.Keys.MessageInput.LaunchEditor:
		messageInput.launchEditorAction()
		return nil
	case config.Keys.MessageInput.Cancel:
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
		data := api.SendMessageData{
			Content:         text,
			Reference:       &discord.MessageReference{MessageID: messagesText.selectedMessage.ID},
			AllowedMentions: &api.AllowedMentions{RepliedUser: option.False},
		}

		if strings.HasPrefix(mi.GetTitle(), "[@]") {
			data.AllowedMentions.RepliedUser = option.True
		}

		go discordState.SendMessageComplex(guildsTree.selectedChannel.ID, data)
	} else {
		go discordState.SendMessage(guildsTree.selectedChannel.ID, text)
	}

	if err != nil {
		log.Println(err)
		return
	}

	messageInput.reset()
}

func (mi *MessageInput) launchEditorAction() {
	e := config.Editor
	if e == "default" {
		e = os.Getenv("EDITOR")
	}

	cmd := exec.Command(e)
	var b strings.Builder
	cmd.Stdout = &b

	app.Suspend(func() {
		err := cmd.Run()
		if err != nil {
			log.Println(err)
			return
		}
	})

	mi.SetText(b.String())
}
