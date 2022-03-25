package ui

import (
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/atotto/clipboard"
	"github.com/ayntgl/astatine"
	"github.com/ayntgl/discordo/discord"
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
	mtv.SetChangedFunc(func() {
		mtv.app.Draw()
	})
	mtv.SetTitleAlign(tview.AlignLeft)
	mtv.SetBorder(true)
	mtv.SetBorderPadding(0, 0, 1, 1)
	mtv.SetInputCapture(mtv.onInputCapture)
	return mtv
}

func (mtv *MessagesTextView) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if mtv.app.SelectedChannel == nil {
		return nil
	}

	ms := mtv.app.SelectedChannel.Messages
	if len(ms) == 0 {
		return nil
	}

	switch e.Name() {
	case mtv.app.Config.Keys.SelectPreviousMessage:
		if len(mtv.app.MessagesTextView.GetHighlights()) == 0 {
			mtv.app.SelectedMessage = len(ms) - 1
		} else {
			mtv.app.SelectedMessage--
			if mtv.app.SelectedMessage < 0 {
				mtv.app.SelectedMessage = 0
			}
		}

		mtv.app.MessagesTextView.
			Highlight(ms[mtv.app.SelectedMessage].ID).
			ScrollToHighlight()

		return nil
	case mtv.app.Config.Keys.SelectNextMessage:
		if len(mtv.app.MessagesTextView.GetHighlights()) == 0 {
			mtv.app.SelectedMessage = len(ms) - 1
		} else {
			mtv.app.SelectedMessage++
			if mtv.app.SelectedMessage >= len(ms) {
				mtv.app.SelectedMessage = len(ms) - 1
			}
		}

		mtv.app.MessagesTextView.
			Highlight(ms[mtv.app.SelectedMessage].ID).
			ScrollToHighlight()

		return nil
	case mtv.app.Config.Keys.SelectFirstMessage:
		mtv.app.SelectedMessage = 0
		mtv.app.MessagesTextView.
			Highlight(ms[mtv.app.SelectedMessage].ID).
			ScrollToHighlight()

		return nil
	case mtv.app.Config.Keys.SelectLastMessage:
		mtv.app.SelectedMessage = len(ms) - 1
		mtv.app.MessagesTextView.
			Highlight(ms[mtv.app.SelectedMessage].ID).
			ScrollToHighlight()

		return nil
	case mtv.app.Config.Keys.OpenMessageActionsList:
		messageActionsList := tview.NewList()

		hs := mtv.app.MessagesTextView.GetHighlights()
		if len(hs) == 0 {
			return nil
		}

		_, m := discord.FindMessageByID(mtv.app.SelectedChannel.Messages, hs[0])
		if m == nil {
			return nil
		}

		if discord.HasPermission(mtv.app.Session.State, mtv.app.SelectedChannel.ID, astatine.PermissionSendMessages) {
			messageActionsList.AddItem("Reply", "", 'r', func() {
				mtv.app.MessageInputField.SetTitle("Replying to " + m.Author.String())
				mtv.app.
					SetRoot(mtv.app.MainFlex, true).
					SetFocus(mtv.app.MessageInputField)
			})

			messageActionsList.AddItem("Mention Reply", "", 'R', func() {
				mtv.app.MessageInputField.SetTitle("[@] Replying to " + m.Author.String())
				mtv.app.
					SetRoot(mtv.app.MainFlex, false).
					SetFocus(mtv.app.MessageInputField)
			})
		}

		if m.ReferencedMessage != nil {
			messageActionsList.AddItem("Select Reply", "", 'm', nil)
		}

		links := linkRegex.FindAllString(m.Content, -1)
		if len(links) != 0 {
			messageActionsList.AddItem("Open Link", "", 'l', func() {
				for _, l := range links {
					go open.Run(l)
				}
			})
		}

		if len(m.Attachments) != 0 {
			messageActionsList.AddItem("Download Attachment", "", 'd', nil)
			messageActionsList.AddItem("Open Attachment", "", 'o', nil)
		}

		messageActionsList.AddItem("Copy Content", "", 'c', func() {
			if err := clipboard.WriteAll(m.Content); err != nil {
				return
			}

			mtv.app.SetRoot(mtv.app.MainFlex, false)
		})

		messageActionsList.AddItem("Copy ID", "", 'i', func() {
			if err := clipboard.WriteAll(m.ID); err != nil {
				return
			}

			mtv.app.SetRoot(mtv.app.MainFlex, true)
		})

		messageActionsList.
			ShowSecondaryText(false).
			SetDoneFunc(func() {
				mtv.app.
					SetRoot(mtv.app.MainFlex, true).
					SetFocus(mtv.app.MessagesTextView)
			}).
			SetSelectedFunc(func(_ int, mainText string, _ string, _ rune) {
				onMessageActionsListSelected(mtv.app, mainText, m)
			}).
			SetTitle("Press the Escape key to close").
			SetBorder(true)

		mtv.app.SetRoot(messageActionsList, true)

		return nil
	case "Esc":
		mtv.app.SelectedMessage = -1
		mtv.app.SetFocus(mtv.app.MainFlex)
		mtv.app.MessagesTextView.
			Clear().
			Highlight()

		return nil
	}

	return e
}

func onMessageActionsListSelected(app *App, mainText string, m *astatine.Message) {
	switch mainText {
	case "Select Reply":
		app.SelectedMessage, _ = discord.FindMessageByID(app.SelectedChannel.Messages, m.ReferencedMessage.ID)
		app.MessagesTextView.
			Highlight(m.ReferencedMessage.ID).
			ScrollToHighlight()
		app.
			SetRoot(app.MainFlex, false).
			SetFocus(app.MessagesTextView)
	case "Download Attachment":
		for _, a := range m.Attachments {
			f, err := os.Create(filepath.Join(app.Config.General.AttachmentDownloadsDir, a.Filename))
			if err != nil {
				f.Close()
				return
			}
			defer f.Close()

			resp, err := http.Get(a.URL)
			if err != nil {
				return
			}

			d, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			f.Write(d)
		}

		app.SetRoot(app.MainFlex, false)
	case "Open Attachment":
		for _, a := range m.Attachments {
			// We are caching the files, but files with the same name can still exist, so it ultimately does not matter
			cachePath, _ := os.UserCacheDir()
			f, err := os.Create(filepath.Join(cachePath, a.Filename))
			if err != nil {
				return
			}
			defer f.Close()

			resp, err := http.Get(a.URL)
			if err != nil {
				return
			}

			d, err := ioutil.ReadAll(resp.Body)
			if err != nil {
				return
			}

			f.Write(d)
			go open.Run(f.Name())
		}

		app.SetRoot(app.MainFlex, false)
	}
}

type MessageInputField struct {
	*tview.InputField
	app *App
}

func NewMessageInputField(app *App) *MessageInputField {
	mi := &MessageInputField{
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

func (mi *MessageInputField) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	switch e.Name() {
	case "Enter":
		if mi.app.SelectedChannel == nil {
			return nil
		}

		t := strings.TrimSpace(mi.app.MessageInputField.GetText())
		if t == "" {
			return nil
		}

		if len(mi.app.MessagesTextView.GetHighlights()) != 0 {
			_, m := discord.FindMessageByID(mi.app.SelectedChannel.Messages, mi.app.MessagesTextView.GetHighlights()[0])
			d := &astatine.MessageSend{
				Content:         t,
				Reference:       m.Reference(),
				AllowedMentions: &astatine.MessageAllowedMentions{RepliedUser: false},
			}
			if strings.HasPrefix(mi.app.MessageInputField.GetTitle(), "[@]") {
				d.AllowedMentions.RepliedUser = true
			} else {
				d.AllowedMentions.RepliedUser = false
			}

			go mi.app.Session.ChannelMessageSendComplex(m.ChannelID, d)

			mi.app.SelectedMessage = -1
			mi.app.MessagesTextView.Highlight()

			mi.app.MessageInputField.SetTitle("")
		} else {
			go mi.app.Session.ChannelMessageSend(mi.app.SelectedChannel.ID, t)
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
