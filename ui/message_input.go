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
	lua "github.com/yuin/gopher-lua"
)

type MessageInput struct {
	*tview.InputField
	core *Core
}

func NewMessageInput(c *Core) *MessageInput {
	mi := &MessageInput{
		InputField: tview.NewInputField(),
		core:       c,
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
	keysTable := mi.core.Config.State.GetGlobal("keys").(*lua.LTable)
	messageInput := keysTable.RawGetString("messageInput")

	switch e.Name() {
	case "Enter":
		return mi.sendMessage()
	case "Ctrl+V":
		return mi.pasteFromClipboard()
	case "Esc":
		mi.
			SetText("").
			SetTitle("")
		mi.core.Application.SetFocus(mi.core.MainFlex)

		mi.core.MessagesPanel.SelectedMessage = -1
		mi.core.MessagesPanel.Highlight()
		return nil
	case mi.core.Config.String(messageInput):
		return mi.openExternalEditor()
	}

	return e
}

func (mi *MessageInput) sendMessage() *tcell.EventKey {
	if mi.core.ChannelsTree.SelectedChannel == nil {
		return nil
	}

	t := strings.TrimSpace(mi.GetText())
	if t == "" {
		return nil
	}

	messagesLimit := mi.core.Config.Number(mi.core.Config.State.GetGlobal("messagesLimit"))
	ms, err := mi.core.State.Messages(mi.core.ChannelsTree.SelectedChannel.ID, uint(messagesLimit))
	if err != nil {
		return nil
	}

	if len(mi.core.MessagesPanel.GetHighlights()) != 0 {
		mID, err := discord.ParseSnowflake(mi.core.MessagesPanel.GetHighlights()[0])
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
		if strings.HasPrefix(mi.GetTitle(), "[@]") {
			d.AllowedMentions.RepliedUser = option.True
		}

		go mi.core.State.SendMessageComplex(m.ChannelID, d)

		mi.core.MessagesPanel.SelectedMessage = -1
		mi.core.MessagesPanel.Highlight()

		mi.SetTitle("")
	} else {
		go mi.core.State.SendMessage(mi.core.ChannelsTree.SelectedChannel.ID, t)
	}

	mi.SetText("")
	return nil
}

func (mi *MessageInput) pasteFromClipboard() *tcell.EventKey {
	text, _ := clipboard.ReadAll()
	text = mi.GetText() + text
	mi.SetText(text)
	return nil
}

func (mi *MessageInput) openExternalEditor() *tcell.EventKey {
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

	mi.core.Application.Suspend(func() {
		err = cmd.Run()
		if err != nil {
			return
		}
	})

	b, err := io.ReadAll(f)
	if err != nil {
		return nil
	}

	mi.SetText(string(b))
	return nil
}
