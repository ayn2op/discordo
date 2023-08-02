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
	*tview.TextArea
}

func newMessageInput() *MessageInput {
	mi := &MessageInput{
		TextArea: tview.NewTextArea(),
	}

	mi.SetTextStyle(tcell.StyleDefault.Background(tcell.GetColor(config.Current.Theme.BackgroundColor)))
	mi.SetClipboard(func(s string) {
		_ = clipboard.WriteAll(s)
	}, func() string {
		text, _ := clipboard.ReadAll()
		return text
	})

	mi.SetInputCapture(mi.onInputCapture)
	mi.SetBackgroundColor(tcell.GetColor(config.Current.Theme.BackgroundColor))

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
	mi.SetText("", true)
}

func (mi *MessageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case config.Current.Keys.MessageInput.Send:
		mi.sendAction()
		return nil
	case "Alt+Enter":
		return tcell.NewEventKey(tcell.KeyEnter, 0, tcell.ModNone)
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

func (mi *MessageInput) launchEditorAction() {
	e := config.Current.Editor
	if e == "default" {
		e = os.Getenv("EDITOR")
	}
	
	// Create a temporary (with discord_msg in the name) file 
	// that we'll open in the editor. The reason is because 
	// capturing Stdout to a variable actually causes editors 
	// to not work for some reason, so we're going with the more
	// reliable method.
	f, err := os.CreateTemp("", config.Name + "-*.md")
	if err != nil {
		log.Println(err)
		return
	}
	f.Close()
	
	// Defer this so the file is deleted when the
	// function returns, regardless of failure or not
	defer os.Remove(f.Name())
	
	cmd := exec.Command(e, f.Name())
	// For some reason, editors won't open without setting
	// these to their os counterparts.
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	app.Suspend(func() {
		err := cmd.Run()
		if err != nil {
			log.Println(err)
			return
		}
	})
	
	// One may ask "Why don't we remove the file earlier?". Well,
	// the file won't contain any message up until this point, and the file
	// is created in /tmp anyway so it'll be cleared on a reboot.
	var msg, read_err = os.ReadFile(f.Name())
	if read_err != nil {
		log.Println(read_err)
		return
	}

	mi.SetText(string(msg), true)
}
