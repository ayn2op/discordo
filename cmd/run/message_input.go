package run

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/config"
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
	mi.SetBackgroundColor(tcell.GetColor(config.Current.Theme.BackgroundColor))
	mi.SetFieldBackgroundColor(tcell.GetColor(config.Current.Theme.BackgroundColor))

	mi.SetTitleColor(tcell.GetColor(config.Current.Theme.TitleColor))
	mi.SetTitleAlign(tview.AlignLeft)

	p := config.Current.Theme.BorderPadding
	mi.SetBorder(config.Current.Theme.Border)
	mi.SetBorderColor(tcell.GetColor(config.Current.Theme.BorderColor))
	mi.SetBorderPadding(p[0], p[1], p[2], p[3])

	return mi
}

func (mi *MessageInput) reset() {
	mi.SetTitle("")
	mi.SetText("")
}

func (mi *MessageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case config.Current.Keys.MessageInput.Send:
		mi.sendAction()
		return nil
	case config.Current.Keys.MessageInput.Paste:
		mi.pasteAction()
		return nil
	case config.Current.Keys.MessageInput.LaunchEditor:
		mainFlex.messageInput.launchEditorAction()
		return nil
	case config.Current.Keys.Cancel:
		mi.reset()
		return nil
	}

	return event
}

func (mi *MessageInput) sendAction() {
	if !mainFlex.guildsTree.selectedChannelID.IsValid() {
		return
	}

	text := strings.TrimSpace(mi.GetText())
	if text == "" {
		return
	}

	if mainFlex.messagesText.selectedMessage != -1 {
		ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
		if err != nil {
			log.Println(err)
			return
		}

		data := api.SendMessageData{
			Content:         text,
			Reference:       &discord.MessageReference{MessageID: ms[mainFlex.messagesText.selectedMessage].ID},
			AllowedMentions: &api.AllowedMentions{RepliedUser: option.False},
		}

		if strings.HasPrefix(mi.GetTitle(), "[@]") {
			data.AllowedMentions.RepliedUser = option.True
		}

		go discordState.SendMessageComplex(mainFlex.guildsTree.selectedChannelID, data)
	} else {
		go discordState.SendMessage(mainFlex.guildsTree.selectedChannelID, text)
	}

	mainFlex.messagesText.selectedMessage = -1
	mainFlex.messagesText.Highlight()
	mi.reset()
}

func (mi *MessageInput) pasteAction() {
	text, err := clipboard.ReadAll()
	if err != nil {
		log.Println(err)
		return
	}

	// Append the text to the message input.
	mi.SetText(mi.GetText() + text)
}

func (mi *MessageInput) launchEditorAction() {
	e := config.Current.Editor
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
