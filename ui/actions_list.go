package ui

import (
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"

	"github.com/atotto/clipboard"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/rivo/tview"
	"github.com/skratchdot/open-golang/open"
)

var linkRegex = regexp.MustCompile("https?://.+")

type ActionsList struct {
	*tview.List
	app     *Application
	message *discord.Message
}

func newActionsList(app *Application, m *discord.Message) *ActionsList {
	v := &ActionsList{
		List:    tview.NewList(),
		app:     app,
		message: m,
	}

	v.ShowSecondaryText(false)
	v.SetDoneFunc(func() {
		app.SetRoot(app.view, true)
		app.SetFocus(app.view.MessagesText)
	})

	isDM := channelIsInDMCategory(app.view.ChannelsTree.selected)

	// If the client user has the `SEND_MESSAGES` permission, add "Reply" and "Mention Reply" actions.
	if isDM || !isDM && hasPermission(app.state, app.view.ChannelsTree.selected.ID, discord.PermissionSendMessages) {
		v.AddItem("Reply", "", 'r', v.replyAction)
		v.AddItem("Mention Reply", "", 'R', v.mentionReplyAction)
	}

	// If the referenced message exists, add a new action to select the reply.
	if m.ReferencedMessage != nil {
		v.AddItem("Select Reply", "", 'm', v.selectReplyAction)
	}

	// If the content of the message contains link(s), add the appropriate actions.
	links := linkRegex.FindAllString(m.Content, -1)
	if len(links) != 0 {
		v.AddItem("Open Link", "", 'l', func() {
			for _, l := range links {
				go open.Run(l)
			}

			app.SetRoot(app.view, true)
			app.SetFocus(app.view.MessagesText)
		})
	}

	// If the message contains attachments, add the appropriate actions to the actions view.
	if len(m.Attachments) != 0 {
		v.AddItem("Open Attachment", "", 'o', v.openAttachmentAction)
		v.AddItem("Download Attachment", "", 'd', v.downloadAttachmentAction)
	}

	me, _ := app.state.MeStore.Me()

	// If the client user has the `MANAGE_MESSAGES` permission, add a new action to delete the message.
	if (isDM && m.Author.ID == me.ID) || (!isDM && hasPermission(app.state, app.view.ChannelsTree.selected.ID, discord.PermissionManageMessages)) {
		v.AddItem("Delete", "", 'd', v.deleteAction)
	}

	v.AddItem("Copy Content", "", 'c', v.copyContentAction)
	v.AddItem("Copy ID", "", 'i', v.copyIDAction)
	v.AddItem("Copy Link", "", 'k', v.copyLinkAction)

	v.SetTitle("Press the Escape key to close")
	v.SetTitleAlign(tview.AlignLeft)
	v.SetBorder(true)
	v.SetBorderPadding(0, 0, 1, 1)

	return v
}

func (al *ActionsList) replyAction() {
	al.app.view.MessageInput.SetTitle("Replying to " + al.message.Author.Tag())

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessageInput)
}

func (al *ActionsList) mentionReplyAction() {
	al.app.view.MessageInput.SetTitle("[@] Replying to " + al.message.Author.Tag())

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessageInput)
}

func (al *ActionsList) selectReplyAction() {
	ms, err := al.app.state.Cabinet.Messages(al.message.ChannelID)
	if err != nil {
		return
	}

	al.app.view.MessagesText.selected, _ = findMessageByID(ms, al.message.ReferencedMessage.ID)
	al.app.view.MessagesText.
		Highlight(al.message.ReferencedMessage.ID.String()).
		ScrollToHighlight()

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessagesText)
}

func (al *ActionsList) openAttachmentAction() {
	for _, a := range al.message.Attachments {
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

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessagesText)
}

func (al *ActionsList) downloadAttachmentAction() {
	for _, a := range al.message.Attachments {
		path, err := os.UserHomeDir()
		if err != nil {
			path = os.TempDir()
		}

		path = filepath.Join(path, "Downloads", a.Filename)
		f, err := os.Create(path)
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

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessagesText)
}

func (al *ActionsList) deleteAction() {
	al.app.view.MessagesText.Clear()

	err := al.app.state.MessageRemove(al.message.ChannelID, al.message.ID)
	if err != nil {
		return
	}

	err = al.app.state.DeleteMessage(al.message.ChannelID, al.message.ID, "Unknown")
	if err != nil {
		return
	}

	// The returned slice will be sorted from latest to oldest.
	ms, err := al.app.state.Cabinet.Messages(al.message.ChannelID)
	if err != nil {
		return
	}

	for i := len(ms) - 1; i >= 0; i-- {
		_, err = al.app.view.MessagesText.Write(buildMessage(al.app, ms[i]))
		if err != nil {
			return
		}
	}

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessagesText)
}

func (al *ActionsList) copyContentAction() {
	err := clipboard.WriteAll(al.message.Content)
	if err != nil {
		return
	}

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessagesText)
}

func (al *ActionsList) copyIDAction() {
	err := clipboard.WriteAll(al.message.ID.String())
	if err != nil {
		return
	}

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessagesText)
}

func (al *ActionsList) copyLinkAction() {
	err := clipboard.WriteAll(al.message.URL())
	if err != nil {
		return
	}

	al.app.SetRoot(al.app.view, true)
	al.app.SetFocus(al.app.view.MessagesText)
}
