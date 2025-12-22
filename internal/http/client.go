package http

import (
	"net/http"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
)

func NewClient(token string) *api.Client {
	stdClient := http.DefaultClient
	stdClient.Transport = NewTransport()
	httpClient := httputil.NewClientWithDriver(httpdriver.WrapClient(*stdClient))
	apiClient := api.NewCustomClient(token, httpClient)
	apiClient.UserAgent = BrowserUserAgent
	return apiClient
}
