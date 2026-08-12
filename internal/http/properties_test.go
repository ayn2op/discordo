package http

import (
	"strings"
	"testing"

	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/tls"
)

func TestBrowserIdentity(t *testing.T) {
	properties := CommonProperties()

	tests := []struct {
		name string
		got  any
		want any
	}{
		{"properties browser", properties[gateway.IdentifyBrowser], tls.BrowserName()},
		{"properties version", properties["browser_version"], tls.BrowserVersion()},
		{"properties user agent", properties["browser_user_agent"], BrowserUserAgent()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if test.got != test.want {
				t.Fatalf("got = %v, want = %v", test.got, test.want)
			}
		})
	}

	if product := tls.BrowserName() + "/" + tls.BrowserVersion(); !strings.Contains(BrowserUserAgent(), product) {
		t.Fatalf("user agent does not contain %q", product)
	}
}

func TestBrowserClientHints(t *testing.T) {
	headers := Headers()
	tests := []struct {
		name string
		want string
	}{
		{"Sec-Ch-Ua", `"Chromium";v="146", "Not-A.Brand";v="24", "Google Chrome";v="146"`},
		{"Sec-Ch-Ua-Mobile", "?0"},
		{"Sec-Ch-Ua-Platform", `"Windows"`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := headers.Get(test.name); got != test.want {
				t.Fatalf("got = %q, want = %q", got, test.want)
			}
		})
	}
}
