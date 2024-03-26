package cmd

import (
	"log"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/constants"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessageInput struct {
	*tview.TextArea
	replyMessageIdx int
}

func newMessageInput() *MessageInput {
	mi := &MessageInput{
		TextArea:        tview.NewTextArea(),
		replyMessageIdx: -1,
	}

	mi.SetTextStyle(tcell.StyleDefault.Background(tcell.GetColor(cfg.Theme.BackgroundColor)))
	mi.SetClipboard(func(s string) {
		_ = clipboard.WriteAll(s)
	}, func() string {
		text, _ := clipboard.ReadAll()
		return text
	})

	mi.SetInputCapture(mi.onInputCapture)
	mi.SetBackgroundColor(tcell.GetColor(cfg.Theme.BackgroundColor))

	mi.SetTitleColor(tcell.GetColor(cfg.Theme.TitleColor))
	mi.SetTitleAlign(tview.AlignLeft)

	p := cfg.Theme.BorderPadding
	mi.SetBorder(cfg.Theme.Border)
	mi.SetBorderColor(tcell.GetColor(cfg.Theme.BorderColor))
	mi.SetBorderPadding(p[0], p[1], p[2], p[3])

	return mi
}

func (mi *MessageInput) reset() {
	mi.SetTitle("")
	mi.SetText("", true)
}

func (mi *MessageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch mainFlex.mode {
	case ModeInsert:
		switch event.Name() {
		case cfg.Keys.Insert.MessageInput.Send:
			if !mainFlex.guildsTree.selectedChannelID.IsValid() {
				return nil
			}

			text := strings.TrimSpace(mi.GetText())
			if text == "" {
				return nil
			}

			if mi.replyMessageIdx != -1 {
				ms, err := discordState.Cabinet.Messages(mainFlex.guildsTree.selectedChannelID)
				if err != nil {
					log.Println(err)
					return nil
				}

				data := api.SendMessageData{
					Content:         text,
					Reference:       &discord.MessageReference{MessageID: ms[mi.replyMessageIdx].ID},
					AllowedMentions: &api.AllowedMentions{RepliedUser: option.False},
				}

				if strings.HasPrefix(mi.GetTitle(), "[@]") {
					data.AllowedMentions.RepliedUser = option.True
				}

				go func() {
					if _, err := discordState.SendMessageComplex(mainFlex.guildsTree.selectedChannelID, data); err != nil {
						log.Println(err)
					}
				}()
			} else {
				go func() {
					if _, err := discordState.SendMessage(mainFlex.guildsTree.selectedChannelID, text); err != nil {
						log.Println(err)
					}
				}()
			}

			mi.replyMessageIdx = -1
			mainFlex.messagesText.Highlight()
			mi.reset()
			return nil
		case cfg.Keys.Insert.MessageInput.Editor:
			e := cfg.Editor
			if e == "default" {
				e = os.Getenv("EDITOR")
			}

			f, err := os.CreateTemp("", constants.TmpFilePattern)
			if err != nil {
				log.Println(err)
				return nil
			}
			_, _ = f.WriteString(mi.GetText())
			f.Close()

			defer os.Remove(f.Name())

			cmd := exec.Command(e, f.Name())
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

			msg, err := os.ReadFile(f.Name())
			if err != nil {
				log.Println(err)
				return nil
			}

			mi.SetText(strings.TrimSpace(string(msg)), true)
			return nil
		}
	}

	return event
}
