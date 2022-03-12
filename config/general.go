package config

import (
	"os"
	"runtime"
)

type IdentifyConfig struct {
	Os      string `toml:"os"`
	Browser string `toml:"browser"`
}

type GeneralConfig struct {
	UserAgent          string         `toml:"user_agent"`
	FetchMessagesLimit int            `toml:"fetch_messages_limit"`
	Mouse              bool           `toml:"mouse"`
	Timestamps         bool           `toml:"timestamps"`
	Identify           IdentifyConfig `toml:"identify"`
	DownloadLocation   string         `toml:"attachment_download_location"`
}

func newGeneralConfig() GeneralConfig {

	// We try to set the download folder location to the default Downloads folder
	dlloc := "~/Downloads/"
	if runtime.GOOS == "windows" {
		h, _ := os.UserHomeDir()
		dlloc = h + "\\Downloads\\"
	} else if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		h, _ := os.UserHomeDir()
		dlloc = h + "/Downloads/"
	} else {
		dlloc = os.TempDir() // Very lame fallback, I know
	}
	return GeneralConfig{
		UserAgent:          "Mozilla/5.0 (X11; Linux x86_64; rv:97.0) Gecko/20100101 Firefox/97.0",
		FetchMessagesLimit: 50,
		Mouse:              true,
		Timestamps:         false,
		DownloadLocation:   dlloc,
		Identify: IdentifyConfig{
			Os:      "Linux",
			Browser: "Firefox",
		},
	}
}
