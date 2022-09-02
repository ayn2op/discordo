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
	luar "layeh.com/gopher-luar"
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
	keysTable, ok := mi.core.Config.State.GetGlobal("keys").(*lua.LTable)
	if !ok {
		keysTable = mi.core.Config.State.NewTable()
	}

	messageInputTable, ok := keysTable.RawGetString("messageInput").(*lua.LTable)
	if !ok {
		messageInputTable = mi.core.Config.State.NewTable()
	}

	var fn lua.LValue
	messageInputTable.ForEach(func(k, v lua.LValue) {
		keyTable := v.(*lua.LTable)
		if e.Name() == lua.LVAsString(keyTable.RawGetString("name")) {
			fn = keyTable.RawGetString("action")
		}
	})

	if fn != nil {
		mi.core.Config.State.CallByParam(lua.P{
			Fn:      fn,
			NRet:    1,
			Protect: true,
		}, luar.New(mi.core.Config.State, mi.core), luar.New(mi.core.Config.State, e))
		// Returned value
		ret, ok := mi.core.Config.State.Get(-1).(*lua.LUserData)
		if !ok {
			return e
		}

		// Remove returned value
		mi.core.Config.State.Pop(1)

		ev, ok := ret.Value.(*tcell.EventKey)
		if ok {
			return ev
		}
	}

	// Defaults
	switch e.Name() {
	case "Enter":
		return mi.sendMessage()
	case "Esc":
		mi.
			SetText("").
			SetTitle("")
		mi.core.MessagesPanel.SelectedMessage = -1
		mi.core.MessagesPanel.Highlight()
		return nil
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

	messagesLimit := mi.core.Config.State.GetGlobal("messagesLimit")
	ms, err := mi.core.State.Messages(mi.core.ChannelsTree.SelectedChannel.ID, uint(lua.LVAsNumber(messagesLimit)))
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

func (mi *MessageInput) pasteClipboardContentLua(s *lua.LState) int {
	text, _ := clipboard.ReadAll()
	text = mi.GetText() + text
	mi.SetText(text)
	return returnNilLua(s)
}

func (mi *MessageInput) openExternalEditorLua(s *lua.LState) int {
	e := os.Getenv("EDITOR")
	if e == "" {
		return returnNilLua(s)
	}

	f, err := os.CreateTemp(os.TempDir(), "discordo-*.md")
	if err != nil {
		return returnNilLua(s)
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
		return returnNilLua(s)
	}

	mi.SetText(string(b))
	return returnNilLua(s)
}
