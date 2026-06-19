package http

import (
	"net/http"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/arikawa/v3/utils/httputil/httpdriver"
)

func NewClient(token string) *api.Client {
	stdClient := new(http.Client)
	stdClient.Transport = NewTransport()
	httpClient := httputil.NewClientWithDriver(httpdriver.WrapClient(*stdClient))
	apiClient := api.NewCustomClient(token, httpClient)
	apiClient.UserAgent = BrowserUserAgent
	return apiClient
}
