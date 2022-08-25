package ui

import (
	"io"
	"os"
	"os/exec"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

type MessageInput struct {
	*tview.InputField
	app *App
}

func NewMessageInput(app *App) *MessageInput {
	mi := &MessageInput{
		InputField: tview.NewInputField(),
		app:        app,
	}

	mi.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	mi.SetPlaceholder("Message...")
	mi.SetPlaceholderStyle(tcell.StyleDefault.Background(tview.Styles.PrimitiveBackgroundColor))
	mi.SetInputCapture(mi.onInputCapture)

	mi.SetTitleAlign(tview.AlignLeft)
	mi.SetBorder(true)
	mi.SetBorderPadding(0, 0, 1, 1)

	return mi
}

func (mi *MessageInput) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case "Enter":
		if mi.app.ChannelsTree.SelectedChannel == nil {
			return nil
		}

		t := strings.TrimSpace(mi.app.MessageInputField.GetText())
		if t == "" {
			return nil
		}

		ms, err := mi.app.State.Messages(mi.app.ChannelsTree.SelectedChannel.ID, mi.app.Config.MessagesLimit)
		if err != nil {
			return nil
		}

		if len(mi.app.MessagesPanel.GetHighlights()) != 0 {
			mID, err := discord.ParseSnowflake(mi.app.MessagesPanel.GetHighlights()[0])
			if err != nil {
				return nil
			}

			_, m := findMessageByID(ms, discord.MessageID(mID))
			d := api.SendMessageData{
				Content:         t,
				Reference:       m.Reference,
				AllowedMentions: &api.AllowedMentions{RepliedUser: option.False},
			}

			// If the title of the message InputField widget has "[@]" as a prefix, send the message as a reply and mention the replied user.
			if strings.HasPrefix(mi.app.MessageInputField.GetTitle(), "[@]") {
				d.AllowedMentions.RepliedUser = option.True
			}

			go mi.app.State.SendMessageComplex(m.ChannelID, d)

			mi.app.MessagesPanel.SelectedMessage = -1
			mi.app.MessagesPanel.Highlight()

			mi.app.MessageInputField.SetTitle("")
		} else {
			go mi.app.State.SendMessage(mi.app.ChannelsTree.SelectedChannel.ID, t)
		}

		mi.app.MessageInputField.SetText("")

		return nil
	case "Ctrl+V":
		text, _ := clipboard.ReadAll()
		text = mi.app.MessageInputField.GetText() + text
		mi.app.MessageInputField.SetText(text)

		return nil
	case "Esc":
		mi.app.MessageInputField.
			SetText("").
			SetTitle("")
		mi.app.SetFocus(mi.app.MainFlex)

		mi.app.MessagesPanel.SelectedMessage = -1
		mi.app.MessagesPanel.Highlight()

		return nil
	case mi.app.Config.Keys.OpenExternalEditor:
		e := os.Getenv("EDITOR")
		if e == "" {
			return nil
		}

		f, err := os.CreateTemp(os.TempDir(), "discordo-*.md")
		if err != nil {
			return nil
		}
		defer os.Remove(f.Name())

		cmd := exec.Command(e, f.Name())
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout

		mi.app.Suspend(func() {
			err = cmd.Run()
			if err != nil {
				return
			}
		})

		b, err := io.ReadAll(f)
		if err != nil {
			return nil
		}

		mi.app.MessageInputField.SetText(string(b))

		return nil
	}

	return e
}
