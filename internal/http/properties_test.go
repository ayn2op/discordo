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
