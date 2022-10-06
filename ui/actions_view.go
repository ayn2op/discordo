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

type ActionsView struct {
	*tview.List
	core    *Core
	message *discord.Message
}

func newActionsView(c *Core, m *discord.Message) *ActionsView {
	v := &ActionsView{
		List:    tview.NewList(),
		core:    c,
		message: m,
	}

	v.ShowSecondaryText(false)
	v.SetDoneFunc(func() {
		c.App.SetRoot(c.View, true)
		c.App.SetFocus(c.MessagesView)
	})

	// If the client user has the `SEND_MESSAGES` permission, add "Reply" and "Mention Reply" actions.
	if hasPermission(c.State, c.ChannelsView.selectedChannel.ID, discord.PermissionSendMessages) {
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

			c.App.SetRoot(c.View, true)
			c.App.SetFocus(c.MessagesView)
		})
	}

	// If the message contains attachments, add the appropriate actions to the actions view.
	if len(m.Attachments) != 0 {
		v.AddItem("Open Attachment", "", 'o', v.openAttachmentAction)
		v.AddItem("Download Attachment", "", 'd', v.downloadAttachmentAction)
	}

	// If the client user has the `MANAGE_MESSAGES` permission, add a new action to delete the message.
	if hasPermission(c.State, c.ChannelsView.selectedChannel.ID, discord.PermissionManageMessages) {
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

func (v *ActionsView) replyAction() {
	v.core.InputView.SetTitle("Replying to " + v.message.Author.Tag())

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.InputView)
}

func (v *ActionsView) mentionReplyAction() {
	v.core.InputView.SetTitle("[@] Replying to " + v.message.Author.Tag())

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.InputView)
}

func (v *ActionsView) selectReplyAction() {
	v.core.MessagesView.
		Highlight(v.message.ReferencedMessage.ID.String()).
		ScrollToHighlight()

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.MessagesView)
}

func (v *ActionsView) openAttachmentAction() {
	for _, a := range v.message.Attachments {
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

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.MessagesView)
}

func (v *ActionsView) downloadAttachmentAction() {
	for _, a := range v.message.Attachments {
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

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.MessagesView)
}

func (v *ActionsView) deleteAction() {
	v.core.MessagesView.Clear()

	err := v.core.State.MessageRemove(v.message.ChannelID, v.message.ID)
	if err != nil {
		return
	}

	err = v.core.State.DeleteMessage(v.message.ChannelID, v.message.ID, "Unknown")
	if err != nil {
		return
	}

	// The returned slice will be sorted from latest to oldest.
	ms, err := v.core.State.Cabinet.Messages(v.message.ChannelID)
	if err != nil {
		return
	}

	for i := len(ms) - 1; i >= 0; i-- {
		_, err = v.core.MessagesView.Write(buildMessage(v.core, ms[i]))
		if err != nil {
			return
		}
	}

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.MessagesView)
}

func (v *ActionsView) copyContentAction() {
	err := clipboard.WriteAll(v.message.Content)
	if err != nil {
		return
	}

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.MessagesView)
}

func (v *ActionsView) copyIDAction() {
	err := clipboard.WriteAll(v.message.ID.String())
	if err != nil {
		return
	}

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.MessagesView)
}

func (v *ActionsView) copyLinkAction() {
	err := clipboard.WriteAll(v.message.URL())
	if err != nil {
		return
	}

	v.core.App.SetRoot(v.core.View, true)
	v.core.App.SetFocus(v.core.MessagesView)
}
