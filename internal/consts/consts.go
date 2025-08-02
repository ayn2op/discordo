package consts

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

const Name = "discordo"

const identifyPropertiesURL = "https://cordapi.dolfi.es/api/v2/properties/web"

var defaultIdentifyProps = gateway.IdentifyProperties{
	Device: "",

	Browser:          "Chrome",
	BrowserUserAgent: "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/138.0.0.0 Safari/537.36",
	BrowserVersion:   "138.0.0.0",

	OS:        "Windows",
	OSVersion: "10",

	ClientBuildNumber: 415522,
	ReleaseChannel:    "stable",

	SystemLocale:  discord.EnglishUS,
	HasClientMods: false,
}

func GetIdentifyProps() gateway.IdentifyProperties {
	resp, err := http.Get(identifyPropertiesURL)
	if err != nil {
		return defaultIdentifyProps
	}
	defer resp.Body.Close()

	var props struct {
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
	if err := json.NewDecoder(resp.Body).Decode(&props); err != nil {
		return defaultIdentifyProps
	}

	return gateway.IdentifyProperties{
		Device: "",

		Browser:          props.Browser.Type,
		BrowserUserAgent: props.Browser.UserAgent,
		BrowserVersion:   props.Browser.Version,

		OS:        props.Browser.OS.Type,
		OSVersion: props.Browser.OS.Version,

		ClientBuildNumber: props.Client.BuildNumber,
		ReleaseChannel:    props.Client.ReleaseChannel,

		SystemLocale:  discord.EnglishUS,
		HasClientMods: false,
	}
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
