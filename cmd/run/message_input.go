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

	ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
	if err != nil {
		log.Println(err)
		return
	}

	substitution := false
	rp := strings.Split(text, "/")
	if strings.HasPrefix(text, "s/") && len(rp) == 3 {
		substitution = true
		for i, msg := range ms {
			if msg.Author.ID == discordState.Ready().User.ID {
				text = strings.ReplaceAll(msg.Content, rp[1], rp[2])
				mainFlex.messagesText.selectedMessage = i
				break
			}
		}
	}

	if mainFlex.messagesText.selectedMessage != -1 {
		title := mi.GetTitle()

		if strings.HasPrefix(title, "Replying") {
			data := api.SendMessageData{
				Content:         text,
				Reference:       &discord.MessageReference{MessageID: ms[mainFlex.messagesText.selectedMessage].ID},
				AllowedMentions: &api.AllowedMentions{RepliedUser: option.False},
			}
	
			if strings.HasSuffix(title, "[@]") {
				data.AllowedMentions.RepliedUser = option.True
			}
	
			go discordState.SendMessageComplex(mainFlex.guildsTree.selectedChannelID, data)
		} else if strings.HasPrefix(title, "Editing") || substitution {
			m, err := discordState.EditText(mainFlex.guildsTree.selectedChannelID, ms[mainFlex.messagesText.selectedMessage].ID, text)
			if err != nil {
				log.Println(err)
				return
			}

			err = discordState.MessageSet(m, true)
			if err != nil {
				log.Println(err)
				return
			}

			redrawChannel(mainFlex.guildsTree.selectedChannelID)
		}

		mainFlex.messagesText.selectedMessage = -1
		mainFlex.messagesText.Highlight()
	} else {
		go discordState.SendMessage(mainFlex.guildsTree.selectedChannelID, text)
	}

	mi.reset()
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

	mi.SetText(b.String(), true)
}
