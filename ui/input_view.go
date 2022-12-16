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

	app *Application
}

func newInputView(app *Application) *InputView {
	v := &InputView{
		InputField: tview.NewInputField(),

		app: app,
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
	case v.app.config.Keys.InputView.SendMessage:
		return v.sendMessage()
	case v.app.config.Keys.InputView.OpenExternalEditor:
		return v.openExternalEditor()
	case v.app.config.Keys.InputView.PasteClipboard:
		return v.pasteClipboard()
	case "Esc":
		v.
			SetText("").
			SetTitle("")
		v.app.view.MessagesView.selected = -1
		v.app.view.MessagesView.Highlight()
		return nil
	}

	return event
}

func (v *InputView) sendMessage() *tcell.EventKey {
	if v.app.view.ChannelsView.selected == nil {
		return nil
	}

	t := strings.TrimSpace(v.GetText())
	if t == "" {
		return nil
	}

	ms, err := v.app.state.Messages(v.app.view.ChannelsView.selected.ID, v.app.config.MessagesLimit)
	if err != nil {
		log.Println(err)
		return nil
	}

	if len(v.app.view.MessagesView.GetHighlights()) != 0 {
		mID, err := discord.ParseSnowflake(v.app.view.MessagesView.GetHighlights()[0])
		if err != nil {
			log.Println(err)
			return nil
		}

		_, m := findMessageByID(ms, discord.MessageID(mID))
		d := api.SendMessageData{
			Content: t,
			Reference: &discord.MessageReference{
				MessageID: m.ID,
			},
			AllowedMentions: &api.AllowedMentions{RepliedUser: option.False},
		}

		// If the title of the input view has "[@]" as a prefix, send the message as a reply and mention the replied user.
		if strings.HasPrefix(v.GetTitle(), "[@]") {
			d.AllowedMentions.RepliedUser = option.True
		}

		go v.app.state.SendMessageComplex(m.ChannelID, d)

		v.app.view.MessagesView.selected = -1
		v.app.view.MessagesView.Highlight()

		v.SetTitle("")
	} else {
		go v.app.state.SendMessage(v.app.view.ChannelsView.selected.ID, t)
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

	v.app.Suspend(func() {
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
