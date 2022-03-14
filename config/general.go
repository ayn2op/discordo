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
	UserAgent              string         `toml:"user_agent"`
	FetchMessagesLimit     int            `toml:"fetch_messages_limit"`
	Mouse                  bool           `toml:"mouse"`
	Timestamps             bool           `toml:"timestamps"`
	Identify               IdentifyConfig `toml:"identify"`
	AttachmentDownloadsDir string         `toml:"attachment_downloads_dir"`
}

func newGeneralConfig() GeneralConfig {
	return GeneralConfig{
		UserAgent:              "Mozilla/5.0 (X11; Linux x86_64; rv:97.0) Gecko/20100101 Firefox/97.0",
		FetchMessagesLimit:     50,
		Mouse:                  true,
		Timestamps:             false,
		AttachmentDownloadsDir: UserDownloadsDir(),
		Identify: IdentifyConfig{
			Os:      "Linux",
			Browser: "Firefox",
		},
	}
}

func UserDownloadsDir() string {
	// We try to set the download folder location to the default Downloads folder
	var dlloc string
	if runtime.GOOS == "windows" {
		h, _ := os.UserHomeDir()
		dlloc = h + "\\Downloads"
	} else if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		h, _ := os.UserHomeDir()
		dlloc = h + "/Downloads"
	} else {
		dlloc = os.TempDir() // Very lame fallback, I know
	}
	return dlloc
}
