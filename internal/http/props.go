package http

import (
	"encoding/base64"
	"encoding/json"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/google/uuid"
)

const (
	Browser          = "Chrome"
	BrowserVersion   = "140.0.0.0"
	BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36"
)

var (
	Locale = discord.EnglishUS
)

func IdentifyProperties() gateway.IdentifyProperties {
	return gateway.IdentifyProperties{
		gateway.IdentifyDevice: "",

		gateway.IdentifyOS: "Windows",
		"os_version":       "10",

		gateway.IdentifyBrowser: Browser,
		"browser_version":       BrowserVersion,
		"browser_user_agent":    BrowserUserAgent,

		"client_build_number":         447677,
		"client_event_source":         nil,
		"client_app_state":            "focused",
		"client_launch_id":            uuid.NewString(),
		"client_heartbeat_session_id": uuid.NewString(),

		"launch_signature": generateLaunchSignature(),
		"system_locale":    Locale,
		"release_channel":  "stable",
		"has_client_mods":  false,

		"referrer":                 "",
		"referrer_current":         "",
		"referring_domain":         "",
		"referring_domain_current": "",

		// These properties are only sent when identifying with the gateway and are not included in the X-Super-Properties header.
		"is_fast_connect":         false,
		"gateway_connect_reasons": "AppSkeleton",
	}
}

func getSuperProps() (string, error) {
	props := IdentifyProperties()
	delete(props, "is_fast_connect")
	delete(props, "gateway_connect_reasons")

	raw, err := json.Marshal(props)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(raw), nil
}

func generateLaunchSignature() string {
	// Discord sets specific bits of the UUID to denote the detection
	// of client mods like Vencord. We don't want that.
	// See <https://docs.discord.food/reference#launch-signature>.
	mask := [16]byte{
		0b11111111, 0b01111111, 0b11101111, 0b11101111,
		0b11110111, 0b11101111, 0b11110111, 0b11111111,
		0b11011111, 0b01111110, 0b11111111, 0b10111111,
		0b11111110, 0b11111111, 0b11110111, 0b11111111,
	}
	id := uuid.New()
	for i, maskByte := range mask {
		id[i] &= maskByte
	}
	return id.String()
}
