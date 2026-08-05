//go:build no_spoof_tls_fingerprint

package http

import (
	"net/http"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/utils/httputil"
	"github.com/ayn2op/arikawa/v3/utils/httputil/httpdriver"
)

func NewClient(token string) *api.Client {
	stdClient := http.Client{Transport: NewTransport()}
	httpClient := httputil.NewClientWithDriver(httpdriver.WrapClient(stdClient))
	client := api.NewCustomClient(token, httpClient)
	client.UserAgent = BrowserUserAgent()
	return client
}
