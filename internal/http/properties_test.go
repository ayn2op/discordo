package http

import (
	"strings"
	"testing"
	"uuid"

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
	version := tls.BrowserMajorVersion()
	tests := []struct {
		name string
		want string
	}{
		{"Sec-Ch-Ua", `"Chromium";v="` + version + `", "Not-A.Brand";v="24", "Google Chrome";v="` + version + `"`},
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

func TestPropertyUUIDs(t *testing.T) {
	properties := XSuperProperties()
	for _, key := range []gateway.IdentifyPropertyKey{"client_launch_id", "client_heartbeat_session_id", "launch_signature"} {
		t.Run(string(key), func(t *testing.T) {
			value, ok := properties[key].(string)
			if !ok {
				t.Fatalf("got %T, want string", properties[key])
			}
			id, err := uuid.Parse(value)
			if err != nil {
				t.Fatal(err)
			}
			if id[6]>>4 != 4 || id[8]>>6 != 2 {
				t.Errorf("got non-v4 UUID %q", id)
			}
		})
	}
}
