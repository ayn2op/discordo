package messageslist

import (
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"os"
	"path/filepath"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
	"github.com/ncruces/zenity"
	"github.com/skratchdot/open-golang/open"
)

// ShowAttachmentsMsg asks to pick one of the attachments and links of the selected message.
type ShowAttachmentsMsg picker.Items

func (m Model) open() tview.Cmd {
	return m.openWith(m.openAttachment)
}

func (m Model) openInBrowser() tview.Cmd {
	return m.openWith(func(attachment discord.Attachment) tview.Cmd { return openURL(attachment.URL) })
}

func (m Model) openWith(openAttachment func(discord.Attachment) tview.Cmd) tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok {
		return nil
	}

	urls := m.selectedURLs()
	switch total := len(urls) + len(selectedMessage.Attachments); {
	case total == 0:
		return nil
	case total > 1:
		return m.showAttachmentsList(urls, selectedMessage.Attachments, openAttachment)
	case len(urls) == 1:
		return openURL(urls[0])
	}

	return openAttachment(selectedMessage.Attachments[0])
}

func (m Model) download() tview.Cmd {
	selectedMessage, ok := m.selectedMessage()
	if !ok || len(selectedMessage.Attachments) == 0 {
		return nil
	}
	if len(selectedMessage.Attachments) == 1 {
		attachment := selectedMessage.Attachments[0]
		return m.confirmAttachment(attachment, m.saveAttachment(attachment))
	}

	items := make(picker.Items, len(selectedMessage.Attachments))
	for i, attachment := range selectedMessage.Attachments {
		items[i] = picker.Item{Text: attachment.Filename, Reference: m.confirmAttachment(attachment, m.saveAttachment(attachment))}
	}
	return showAttachmentsPicker(items)
}

func (m Model) showAttachmentsList(urls []string, attachments []discord.Attachment, openAttachment func(discord.Attachment) tview.Cmd) tview.Cmd {
	var items picker.Items
	for _, attachment := range attachments {
		items = append(items, picker.Item{Text: attachment.Filename, Reference: openAttachment(attachment)})
	}
	for _, url := range urls {
		items = append(items, picker.Item{Text: url, Reference: openURL(url)})
	}
	return showAttachmentsPicker(items)
}

func showAttachmentsPicker(items picker.Items) tview.Cmd {
	return func() tview.Msg { return ShowAttachmentsMsg(items) }
}

func (m Model) openAttachment(attachment discord.Attachment) tview.Cmd {
	return m.confirmAttachment(attachment, m.openDownloadedAttachment(attachment))
}

func (m Model) confirmAttachment(attachment discord.Attachment, action tview.Cmd) tview.Cmd {
	if !m.cfg.UI.MessagesList.Attachments.AllowedMIMETypes.Has(attachment.ContentType) {
		return ui.ShowDialog(
			"This attachment type is not allowed and may be unsafe. Continue anyway?",
			ui.DialogButton{Label: "No"},
			ui.DialogButton{Label: "Yes", Cmd: action},
		)
	}
	return action
}

// attachmentExtension returns the extension of the attachment's file name, or else one of its content type.
func attachmentExtension(attachment discord.Attachment) string {
	if extension := filepath.Ext(attachment.Filename); extension != "" {
		return extension
	}

	mediaType, _, _ := mime.ParseMediaType(attachment.ContentType)
	if extensions, _ := mime.ExtensionsByType(mediaType); len(extensions) != 0 {
		return extensions[0]
	}

	return ""
}

func (m Model) openDownloadedAttachment(attachment discord.Attachment) tview.Cmd {
	return func() tview.Msg {
		dir := filepath.Join(consts.CacheDir(), "attachments")
		if err := os.MkdirAll(dir, 0o700); err != nil {
			return ui.ErrorDialog("create attachments dir", err)
		}

		file, err := os.CreateTemp(dir, "attachment-*"+attachmentExtension(attachment))
		if err != nil {
			return ui.ErrorDialog("create attachment file", err)
		}
		path := file.Name()
		if err := file.Close(); err != nil {
			return ui.ErrorDialog("create attachment file", err)
		}

		if err := m.downloadAttachment(attachment, path); err != nil {
			return ui.ErrorDialog("download attachment", errors.Join(err, os.Remove(path)))
		}

		if err := open.Start(path); err != nil {
			return ui.ErrorDialog("open attachment file", err)
		}

		return nil
	}
}

func (m Model) saveAttachment(attachment discord.Attachment) tview.Cmd {
	return func() tview.Msg {
		destination, err := zenity.SelectFileSave(zenity.Filename(filepath.Base(attachment.Filename)), zenity.ConfirmOverwrite())
		if errors.Is(err, zenity.ErrCanceled) {
			return nil
		}
		if err != nil {
			return ui.ErrorDialog("select attachment destination", err)
		}

		if err := m.downloadAttachment(attachment, destination); err != nil {
			return ui.ErrorDialog("download attachment", err)
		}
		return nil
	}
}

func openURL(url string) tview.Cmd {
	return func() tview.Msg {
		if err := open.Start(url); err != nil {
			return ui.ErrorDialog("open URL", err)
		}
		return nil
	}
}

// Get requests the file at url, by a new link to it if that one expired.
func (m Model) Get(url string) (*http.Response, error) {
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode == http.StatusOK {
		return resp, err
	}
	resp.Body.Close() //nolint:errcheck

	var links struct {
		URLs []struct {
			Refreshed string `json:"refreshed"`
		} `json:"refreshed_urls"`
	}
	body := map[string][]string{"attachment_urls": {url}}
	if err := m.state.RequestJSONInto(&links, http.MethodPost, api.Endpoint+"attachments/refresh-urls", httputil.WithJSONBody(body)); err != nil {
		return nil, err
	}

	if len(links.URLs) != 0 {
		if resp, err = http.Get(links.URLs[0].Refreshed); err != nil || resp.StatusCode == http.StatusOK {
			return resp, err
		}
		resp.Body.Close() //nolint:errcheck
	}

	return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
}

func (m Model) downloadAttachment(attachment discord.Attachment, destination string) error {
	resp, err := m.Get(attachment.URL)
	if err != nil {
		return err
	}
	defer resp.Body.Close() //nolint:errcheck
	file, err := os.Create(destination)
	if err != nil {
		return err
	}
	_, err = io.Copy(file, resp.Body)
	return errors.Join(err, file.Close())
}
