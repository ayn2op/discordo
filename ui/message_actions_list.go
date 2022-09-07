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

type MessageActionsList struct {
	*tview.List
	core    *Core
	message *discord.Message
}

func NewMessageActionsList(c *Core, m *discord.Message) *MessageActionsList {
	mal := &MessageActionsList{
		List:    tview.NewList(),
		core:    c,
		message: m,
	}

	mal.ShowSecondaryText(false)
	mal.SetDoneFunc(func() {
		c.Application.SetRoot(c.MainFlex, true)
		c.Application.SetFocus(c.MessagesPanel)
	})

	// If the client user has the `SEND_MESSAGES` permission, add "Reply" and "Mention Reply" actions.
	if hasPermission(c.State, c.ChannelsTree.SelectedChannel.ID, discord.PermissionSendMessages) {
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

			c.Application.SetRoot(c.MainFlex, true)
			c.Application.SetFocus(c.MessagesPanel)
		})
	}

	// If the message contains attachments, add the appropriate actions to the actions list.
	if len(m.Attachments) != 0 {
		mal.AddItem("Open Attachment", "", 'o', mal.openAttachmentAction)
		mal.AddItem("Download Attachment", "", 'd', mal.downloadAttachmentAction)
	}

	// If the client user has the `MANAGE_MESSAGES` permission, add a new action to delete the message.
	if hasPermission(c.State, c.ChannelsTree.SelectedChannel.ID, discord.PermissionManageMessages) {
		mal.AddItem("Delete", "", 'd', mal.deleteAction)
	}

	mal.AddItem("Copy Content", "", 'c', mal.copyContentAction)
	mal.AddItem("Copy ID", "", 'i', mal.copyIDAction)
	mal.AddItem("Copy Link", "", 'k', mal.copyLinkAction)

	mal.SetTitle("Press the Escape key to close")
	mal.SetTitleAlign(tview.AlignLeft)
	mal.SetBorder(true)
	mal.SetBorderPadding(0, 0, 1, 1)

	return mal
}

func (mal *MessageActionsList) replyAction() {
	mal.core.MessageInput.SetTitle("Replying to " + mal.message.Author.Tag())

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessageInput)
}

func (mal *MessageActionsList) mentionReplyAction() {
	mal.core.MessageInput.SetTitle("[@] Replying to " + mal.message.Author.Tag())

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessageInput)
}

func (mal *MessageActionsList) selectReplyAction() {
	ms, err := mal.core.State.Cabinet.Messages(mal.message.ChannelID)
	if err != nil {
		return
	}

	mal.core.MessagesPanel.SelectedMessage, _ = findMessageByID(ms, mal.message.ReferencedMessage.ID)
	mal.core.MessagesPanel.
		Highlight(mal.message.ReferencedMessage.ID.String()).
		ScrollToHighlight()

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessagesPanel)
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

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessagesPanel)
}

func (mal *MessageActionsList) downloadAttachmentAction() {
	for _, a := range mal.message.Attachments {
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

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessagesPanel)
}

func (mal *MessageActionsList) deleteAction() {
	mal.core.MessagesPanel.Clear()

	err := mal.core.State.MessageRemove(mal.message.ChannelID, mal.message.ID)
	if err != nil {
		return
	}

	err = mal.core.State.DeleteMessage(mal.message.ChannelID, mal.message.ID, "Unknown")
	if err != nil {
		return
	}

	// The returned slice will be sorted from latest to oldest.
	ms, err := mal.core.State.Cabinet.Messages(mal.message.ChannelID)
	if err != nil {
		return
	}

	for i := len(ms) - 1; i >= 0; i-- {
		_, err = mal.core.MessagesPanel.Write(buildMessage(mal.core, ms[i]))
		if err != nil {
			return
		}
	}

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessagesPanel)
}

func (mal *MessageActionsList) copyContentAction() {
	err := clipboard.WriteAll(mal.message.Content)
	if err != nil {
		return
	}

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessagesPanel)
}

func (mal *MessageActionsList) copyIDAction() {
	err := clipboard.WriteAll(mal.message.ID.String())
	if err != nil {
		return
	}

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessagesPanel)
}

func (mal *MessageActionsList) copyLinkAction() {
	err := clipboard.WriteAll(mal.message.URL())
	if err != nil {
		return
	}

	mal.core.Application.SetRoot(mal.core.MainFlex, true)
	mal.core.Application.SetFocus(mal.core.MessagesPanel)
}
