package messageslist

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
)

// get requests the file at url, by a new link to it if that one expired.
func (m Model) get(url string) (*http.Response, error) {
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
	resp, err := m.get(attachment.URL)
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
