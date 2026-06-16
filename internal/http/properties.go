package http

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/google/uuid"
)

//go:generate go run generator.go

const (
	OS        = "Windows"
	OSVersion = "10"

	Browser          = "Chrome"
	BrowserVersion   = "143.0.0.0"
	BrowserUserAgent = "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) " + Browser + "/" + BrowserVersion + " Safari/537.36"

	Locale = discord.EnglishUS
)

func CommonProperties() gateway.IdentifyProperties {
	return gateway.IdentifyProperties{
		gateway.IdentifyDevice: "",

		gateway.IdentifyOS: OS,
		"os_version":       OSVersion,

		gateway.IdentifyBrowser: Browser,
		"browser_version":       BrowserVersion,
		"browser_user_agent":    BrowserUserAgent,

		"client_build_number": ClientBuildNumber,
		"client_event_source": nil,
		"client_launch_id":    uuid.NewString(),

		"system_locale":   Locale,
		"release_channel": "stable",
		"has_client_mods": false,

		"referrer":                 "",
		"referrer_current":         "",
		"referring_domain":         "",
		"referring_domain_current": "",
	}
}

func IdentifyProperties() gateway.IdentifyProperties {
	props := CommonProperties()
	props["is_fast_connect"] = true
	return props
}

func XSuperProperties() gateway.IdentifyProperties {
	props := CommonProperties()
	props["client_app_state"] = "focused"
	props["client_heartbeat_session_id"] = uuid.NewString()
	props["launch_signature"] = generateLaunchSignature()
	return props
}

func generateLaunchSignature() string {
	// Discord uses specific UUID bits to detect client modifications.
	// This mask clears detection bits to avoid identification.
	// Reference: https://docs.discord.food/reference#launch-signature
	//
	// Required version and variant bits for UUIDv4 validity are set by google/uuid.
	// Reference: https://github.com/google/uuid/blob/master/version4.go#L54
	mask := [16]byte{
		0b11111111, 0b01111111, 0b11101111, 0b11101111,
		0b11110111, 0b11101111, 0b11110111, 0b11111111,
		0b11011111, 0b01111110, 0b11111111, 0b10111111,
		0b11111110, 0b11111111, 0b11110111, 0b11111111,
	}
	id := uuid.New()
	for i := range mask {
		id[i] &= mask[i]
	}
	return id.String()
}
