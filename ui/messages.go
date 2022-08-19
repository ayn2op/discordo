package ui

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
)

var linkRegex = regexp.MustCompile("https?://.+")

type MessagesTextView struct {
	*tview.TextView
	app *App
}

func NewMessagesTextView(app *App) *MessagesTextView {
	mtv := &MessagesTextView{
		TextView: tview.NewTextView(),
		app:      app,
	}

	mtv.SetDynamicColors(true)
	mtv.SetRegions(true)
	mtv.SetWordWrap(true)
	mtv.SetInputCapture(mtv.onInputCapture)
	mtv.SetChangedFunc(func() {
		mtv.app.Draw()
	})

	mtv.SetTitleAlign(tview.AlignLeft)
	mtv.SetBorder(true)
	mtv.SetBorderPadding(0, 0, 1, 1)

	return mtv
}

func (mtv *MessagesTextView) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if mtv.app.SelectedChannel == nil {
		return nil
	}

	// Messages should return messages ordered from latest to earliest.
	ms, err := mtv.app.State.Cabinet.Messages(mtv.app.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return nil
	}

	switch e.Name() {
	case mtv.app.Config.Keys.SelectPreviousMessage:
		// If there are no highlighted regions, select the latest (last) message in the messages TextView.
		if len(mtv.app.MessagesTextView.GetHighlights()) == 0 {
			mtv.app.SelectedMessage = 0
		} else {
			// If the selected message is the oldest (first) message, select the latest (last) message in the messages TextView.
			if mtv.app.SelectedMessage == len(ms)-1 {
				mtv.app.SelectedMessage = 0
			} else {
				mtv.app.SelectedMessage++
			}
		}

		mtv.app.MessagesTextView.
			Highlight(ms[mtv.app.SelectedMessage].ID.String()).
			ScrollToHighlight()
		return nil
	case mtv.app.Config.Keys.SelectNextMessage:
		// If there are no highlighted regions, select the latest (last) message in the messages TextView.
		if len(mtv.app.MessagesTextView.GetHighlights()) == 0 {
			mtv.app.SelectedMessage = 0
		} else {
			// If the selected message is the latest (last) message, select the oldest (first) message in the messages TextView.
			if mtv.app.SelectedMessage == 0 {
				mtv.app.SelectedMessage = len(ms) - 1
			} else {
				mtv.app.SelectedMessage--
			}
		}

		mtv.app.MessagesTextView.
			Highlight(ms[mtv.app.SelectedMessage].ID.String()).
			ScrollToHighlight()
		return nil
	case mtv.app.Config.Keys.SelectFirstMessage:
		mtv.app.SelectedMessage = len(ms) - 1
		mtv.app.MessagesTextView.
			Highlight(ms[mtv.app.SelectedMessage].ID.String()).
			ScrollToHighlight()
		return nil
	case mtv.app.Config.Keys.SelectLastMessage:
		mtv.app.SelectedMessage = 0
		mtv.app.MessagesTextView.
			Highlight(ms[mtv.app.SelectedMessage].ID.String()).
			ScrollToHighlight()
		return nil
	case mtv.app.Config.Keys.OpenMessageActionsList:
		hs := mtv.app.MessagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		mID, err := discord.ParseSnowflake(hs[0])
		if err != nil {
			return nil
		}

		_, m := findMessageByID(ms, discord.MessageID(mID))
		if m == nil {
			return nil
		}

		actionsList := NewMessageActionsList(mtv.app, m)
		mtv.app.SetRoot(actionsList, true)
		return nil
	case "Esc":
		mtv.app.SelectedMessage = -1
		mtv.app.SetFocus(mtv.app.MainFlex)
		mtv.app.MessagesTextView.
			Clear().
			Highlight().
			SetTitle("")
		return nil
	}

	return e
}

type MessageActionsList struct {
	*tview.List
	app     *App
	message *discord.Message
}

func NewMessageActionsList(app *App, m *discord.Message) *MessageActionsList {
	mal := &MessageActionsList{
		List:    tview.NewList(),
		app:     app,
		message: m,
	}

	mal.ShowSecondaryText(false)
	mal.SetDoneFunc(func() {
		app.
			SetRoot(app.MainFlex, true).
			SetFocus(app.MessagesTextView)
	})

	// If the client user has the `SEND_MESSAGES` permission, add "Reply" and "Mention Reply" actions.
	if hasPermission(app.State, app.SelectedChannel.ID, discord.PermissionSendMessages) {
		mal.AddItem("Reply", "", 'r', mal.replyAction)
		mal.AddItem("Mention Reply", "", 'R', mal.mentionReplyAction)
	}

	// If the referenced message exists, add a new action to select the reply.
	if m.ReferencedMessage != nil {
		mal.AddItem("Select Reply", "", 'm', mal.selectReplyAction)
	}

	// If the content of the message contains link(s), add the appropriate actions to the list.
	links := linkRegex.FindAllString(m.Content, -1)
	if len(links) != 0 {
		mal.AddItem("Open Link", "", 'l', func() {
			for _, l := range links {
				go open.Run(l)
			}

			app.SetRoot(app.MainFlex, true)
			app.SetFocus(app.MessagesTextView)
		})
	}

	// If the message contains attachments, add the appropriate actions to the actions list.
	if len(m.Attachments) != 0 {
		mal.AddItem("Open Attachment", "", 'o', mal.openAttachmentAction)
		mal.AddItem("Download Attachment", "", 'd', mal.downloadAttachmentAction)
	}

	// If the client user has the `MANAGE_MESSAGES` permission, add a new action to delete the message.
	if hasPermission(app.State, app.SelectedChannel.ID, discord.PermissionManageMessages) {
		mal.AddItem("Delete", "", 'd', mal.deleteAction)
	}

	mal.AddItem("Copy Content", "", 'c', mal.copyContentAction)
	mal.AddItem("Copy ID", "", 'i', mal.copyIDAction)

	mal.SetTitle("Press the Escape key to close")
	mal.SetTitleAlign(tview.AlignLeft)
	mal.SetBorder(true)
	mal.SetBorderPadding(0, 0, 1, 1)

	return mal
}

func (mal *MessageActionsList) replyAction() {
	mal.app.MessageInputField.SetTitle("Replying to " + mal.message.Author.Tag())

	mal.app.
		SetRoot(mal.app.MainFlex, true).
		SetFocus(mal.app.MessageInputField)
}

func (mal *MessageActionsList) mentionReplyAction() {
	mal.app.MessageInputField.SetTitle("[@] Replying to " + mal.message.Author.Tag())

	mal.app.
		SetRoot(mal.app.MainFlex, true).
		SetFocus(mal.app.MessageInputField)
}

func (mal *MessageActionsList) selectReplyAction() {
	ms, err := mal.app.State.Cabinet.Messages(mal.message.ChannelID)
	if err != nil {
		return
	}

	mal.app.SelectedMessage, _ = findMessageByID(ms, mal.message.ReferencedMessage.ID)
	mal.app.MessagesTextView.
		Highlight(mal.message.ReferencedMessage.ID.String()).
		ScrollToHighlight()

	mal.app.
		SetRoot(mal.app.MainFlex, true).
		SetFocus(mal.app.MessagesTextView)
}

func (mal *MessageActionsList) openAttachmentAction() {
	for _, a := range mal.message.Attachments {
		cacheDirPath, _ := os.UserCacheDir()
		f, err := os.Create(filepath.Join(cacheDirPath, a.Filename))
		if err != nil {
			return
		}
		defer f.Close()

		resp, err := http.Get(a.URL)
		if err != nil {
			return
		}

		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		f.Write(d)
		go open.Run(f.Name())
	}

	mal.app.
		SetRoot(mal.app.MainFlex, true).
		SetFocus(mal.app.MessagesTextView)
}

func (mal *MessageActionsList) downloadAttachmentAction() {
	for _, a := range mal.message.Attachments {
		f, err := os.Create(filepath.Join(mal.app.Config.AttachmentDownloadsDir, a.Filename))
		if err != nil {
			return
		}
		defer f.Close()

		resp, err := http.Get(a.URL)
		if err != nil {
			return
		}

		d, err := io.ReadAll(resp.Body)
		if err != nil {
			return
		}

		f.Write(d)
	}

	mal.app.
		SetRoot(mal.app.MainFlex, true).
		SetFocus(mal.app.MessagesTextView)
}

func (mal *MessageActionsList) deleteAction() {
	mal.app.MessagesTextView.Clear()

	err := mal.app.State.MessageRemove(mal.message.ChannelID, mal.message.ID)
	if err != nil {
		return
	}

	err = mal.app.State.DeleteMessage(mal.message.ChannelID, mal.message.ID, "Unknown")
	if err != nil {
		return
	}

	// The returned slice will be sorted from latest to oldest.
	ms, err := mal.app.State.Cabinet.Messages(mal.message.ChannelID)
	if err != nil {
		return
	}

	for i := len(ms) - 1; i >= 0; i-- {
		_, err = mal.app.MessagesTextView.Write(buildMessage(mal.app, ms[i]))
		if err != nil {
			return
		}
	}

	mal.app.
		SetRoot(mal.app.MainFlex, true).
		SetFocus(mal.app.MessagesTextView)
}

func (mal *MessageActionsList) copyContentAction() {
	err := clipboard.WriteAll(mal.message.Content)
	if err != nil {
		return
	}

	mal.app.SetRoot(mal.app.MainFlex, true)
	mal.app.SetFocus(mal.app.MessagesTextView)
}

func (mal *MessageActionsList) copyIDAction() {
	err := clipboard.WriteAll(mal.message.ID.String())
	if err != nil {
		return
	}

	mal.app.SetRoot(mal.app.MainFlex, true)
	mal.app.SetFocus(mal.app.MessagesTextView)
}

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

	mi.SetTitleAlign(tview.AlignLeft)
	mi.SetBorder(true)
	mi.SetBorderPadding(0, 0, 1, 1)
	mi.SetInputCapture(mi.onInputCapture)

	return mi
}

func (mi *MessageInput) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case "Enter":
		if mi.app.SelectedChannel == nil {
			return nil
		}

		t := strings.TrimSpace(mi.app.MessageInputField.GetText())
		if t == "" {
			return nil
		}

		ms, err := mi.app.State.Messages(mi.app.SelectedChannel.ID, mi.app.Config.MessagesLimit)
		if err != nil {
			return nil
		}

		if len(mi.app.MessagesTextView.GetHighlights()) != 0 {
			mID, err := discord.ParseSnowflake(mi.app.MessagesTextView.GetHighlights()[0])
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

			mi.app.SelectedMessage = -1
			mi.app.MessagesTextView.Highlight()

			mi.app.MessageInputField.SetTitle("")
		} else {
			go mi.app.State.SendMessage(mi.app.SelectedChannel.ID, t)
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

		mi.app.SelectedMessage = -1
		mi.app.MessagesTextView.Highlight()

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
