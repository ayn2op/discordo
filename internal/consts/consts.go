package consts

import (
	"encoding/json"
	"net/http"

	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/gateway"
)

const Name = "discordo"

const identifyPropertiesURL = "https://cordapi.dolfi.es/api/v2/properties/web"

func GetIdentifyProperties() (*gateway.IdentifyProperties, error) {
	resp, err := http.Get(identifyPropertiesURL)
	if err != nil {
		return nil, err
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
		return nil, err
	}

	return &gateway.IdentifyProperties{
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
	}, nil
}
