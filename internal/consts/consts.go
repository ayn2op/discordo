package consts

import (
	"encoding/json"
	"log/slog"
	"maps"
	"net/http"
	"os"
	"path/filepath"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
	"github.com/google/uuid"
)

const Name = "discordo"

const identifyPropertiesURL = "https://cordapi.dolfi.es/api/v2/properties/web"

var defaultIdentifyProps = gateway.IdentifyProperties{
	gateway.IdentifyDevice: "",

	gateway.IdentifyBrowser: "Chrome",
	"browser_version":       "140.0.0.0",
	"browser_user_agent":    "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/140.0.0.0 Safari/537.36",

	gateway.IdentifyOS: "Windows",
	"os_version":       "10",

	"client_build_number": 439729,
	"client_event_source": nil,
	"client_launch_id":    uuid.NewString(),
	"client_app_state":    "focused",

	"launch_signature":        uuid.NewString(),
	"system_locale":           discord.EnglishUS,
	"release_channel":         "stable",
	"has_client_mods":         false,
	"is_fast_connect":         false,
	"gateway_connect_reasons": "AppSkeleton",

	"referrer":                 "",
	"referrer_current":         "",
	"referring_domain":         "",
	"referring_domain_current": "",
}

type Properties struct {
	Client struct {
		Type           string `json:"type"`
		BuildNumber    int    `json:"build_number"`
		BuildHash      string `json:"build_hash"`
		ReleaseChannel string `json:"release_channel"`
	} `json:"client"`

	Browser struct {
		Type      string `json:"type"`
		UserAgent string `json:"user_agent"`
		Version   string `json:"version"`
		OS        struct {
			Type    string `json:"type"`
			Version string `json:"version"`
		} `json:"os"`
	} `json:"browser"`
}

func GetIdentifyProps() gateway.IdentifyProperties {
	resp, err := http.Get(identifyPropertiesURL)
	if err != nil {
		return defaultIdentifyProps
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return defaultIdentifyProps
	}

	var props Properties
	if err := json.NewDecoder(resp.Body).Decode(&props); err != nil {
		return defaultIdentifyProps
	}

	p := maps.Clone(defaultIdentifyProps)
	p[gateway.IdentifyBrowser] = props.Browser.Type
	p["browser_version"] = props.Browser.Version
	p["browser_user_agent"] = props.Browser.UserAgent

	p[gateway.IdentifyOS] = props.Browser.OS.Type
	p["os_version"] = props.Browser.OS.Version

	p["release_channel"] = props.Client.ReleaseChannel
	p["client_build_number"] = props.Client.BuildNumber
	return p
}

var cacheDir string

func CacheDir() string {
	return cacheDir
}

func init() {
	userCacheDir, err := os.UserCacheDir()
	if err != nil {
		userCacheDir = os.TempDir()
		slog.Warn("failed to get user cache dir; falling back to temp dir", "err", err, "path", userCacheDir)
	}

	cacheDir = filepath.Join(userCacheDir, Name)
	if err := os.MkdirAll(cacheDir, os.ModePerm); err != nil {
		slog.Error("failed to create cache dir", "err", err, "path", cacheDir)
	}
}
