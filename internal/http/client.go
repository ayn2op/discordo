package http

import (
	"net/http"

	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/utils/httputil"
	"github.com/diamondburned/arikawa/v3/utils/httputil/httpdriver"
)

func NewClient(token string) *api.Client {
	httpClient := httputil.NewClient()
	stdClient := http.DefaultClient
	stdClient.Transport = NewTransport()
	httpClient.Client = httpdriver.WrapClient(*stdClient)
	return api.NewCustomClient(token, httpClient)
}
