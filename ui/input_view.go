package ui

import (
	"io"
	"log"
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

type InputView struct {
	*tview.InputField
	core *Core
}

func newInputView(c *Core) *InputView {
	v := &InputView{
		InputField: tview.NewInputField(),
		core:       c,
	}

	v.SetFieldBackgroundColor(tview.Styles.PrimitiveBackgroundColor)
	v.SetPlaceholder("Message...")
	v.SetPlaceholderStyle(tcell.StyleDefault.Background(tview.Styles.PrimitiveBackgroundColor))
	v.SetInputCapture(v.inputCapture)

	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.SetBorderPadding(0, 0, 1, 1)

	return v
}

func (v *InputView) inputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case "Enter":
		return v.sendMessage()
	case v.core.Config.Keys.InputView.OpenExternalEditor:
		return v.openExternalEditor()
	case v.core.Config.Keys.InputView.PasteClipboard:
		return v.pasteClipboard()
	case "Esc":
		v.
			SetText("").
			SetTitle("")
		v.core.MessagesView.selectedMessage = -1
		v.core.MessagesView.Highlight()
		return nil
	}

	return event
}

func (v *InputView) sendMessage() *tcell.EventKey {
	if v.core.ChannelsView.selectedChannel == nil {
		return nil
	}

	t := strings.TrimSpace(v.GetText())
	if t == "" {
		return nil
	}

	ms, err := v.core.State.Messages(v.core.ChannelsView.selectedChannel.ID, v.core.Config.MessagesLimit)
	if err != nil {
		log.Println(err)
		return nil
	}

	if len(v.core.MessagesView.GetHighlights()) != 0 {
		mID, err := discord.ParseSnowflake(v.core.MessagesView.GetHighlights()[0])
		if err != nil {
			log.Println(err)
			return nil
		}

		_, m := findMessageByID(ms, discord.MessageID(mID))
		d := api.SendMessageData{
			Content:         t,
			Reference:       &discord.MessageReference{
				MessageID: discord.MessageID(mID),
			},
			AllowedMentions: &api.AllowedMentions{RepliedUser: option.False},
		}

		// If the title of the input view has "[@]" as a prefix, send the message as a reply and mention the replied user.
		if strings.HasPrefix(v.GetTitle(), "[@]") {
			d.AllowedMentions.RepliedUser = option.True
		}

		go v.core.State.SendMessageComplex(m.ChannelID, d)

		v.core.MessagesView.selectedMessage = -1
		v.core.MessagesView.Highlight()

		v.SetTitle("")
	} else {
		go v.core.State.SendMessage(v.core.ChannelsView.selectedChannel.ID, t)
	}

	v.SetText("")
	return nil
}

func (v *InputView) pasteClipboard() *tcell.EventKey {
	text, err := clipboard.ReadAll()
	if err != nil {
		log.Println(err)
		return nil
	}

	text = v.GetText() + text
	v.SetText(text)
	return nil
}

func (v *InputView) openExternalEditor() *tcell.EventKey {
	e := os.Getenv("EDITOR")
	if e == "" {
		log.Println("environment variable EDITOR is empty")
		return nil
	}

	f, err := os.CreateTemp(os.TempDir(), "discordo-*.md")
	if err != nil {
		log.Println(err)
		return nil
	}
	defer os.Remove(f.Name())

	cmd := exec.Command(e, f.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout

	v.core.App.Suspend(func() {
		err = cmd.Run()
		if err != nil {
			log.Println(err)
			return
		}
	})

	b, err := io.ReadAll(f)
	if err != nil {
		log.Println(err)
		return nil
	}

	v.SetText(string(b))
	return nil
}
