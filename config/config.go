package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

type IdentifyConfig struct {
	UserAgent      string `toml:"user_agent"`
	Browser        string `toml:"browser"`
	BrowserVersion string `toml:"browser_version"`
	Os             string `toml:"os"`
}

type KeysConfig struct {
	ToggleGuildsTree    string `toml:"toggle_guilds_tree"`
	ToggleChannelsTree  string `toml:"toggle_channels_tree"`
	ToggleMessagesPanel string `toml:"toggle_messages_panel"`
	ToggleMessageInput  string `toml:"toggle_message_input"`

	OpenMessageActionsList string `toml:"open_message_actions_list"`
	OpenExternalEditor     string `toml:"open_external_editor"`

	SelectPreviousMessage string `toml:"select_previous_message"`
	SelectNextMessage     string `toml:"select_next_message"`
	SelectFirstMessage    string `toml:"select_first_message"`
	SelectLastMessage     string `toml:"select_last_message"`
}

type ThemeConfig struct {
	Background string `toml:"background"`
	Border     string `toml:"border"`
	Title      string `toml:"title"`
}

type Config struct {
	Mouse                  bool           `toml:"mouse"`
	Timestamps             bool           `toml:"timestamps"`
	MessagesLimit          uint           `toml:"messages_limit"`
	Timezone               string         `toml:"timezone"`
	TimeFormat             string         `toml:"time_format"`
	AttachmentDownloadsDir string         `toml:"attachment_downloads_dir"`
	Identify               IdentifyConfig `toml:"identify"`
	Theme                  ThemeConfig    `toml:"theme"`
	Keys                   KeysConfig     `toml:"keys"`
}

func New() *Config {
	return &Config{
		Mouse:                  true,
		Timestamps:             false,
		MessagesLimit:          50,
		Timezone:               "Local",
		TimeFormat:             time.Stamp,
		AttachmentDownloadsDir: UserDownloadsDir(),
		Identify: IdentifyConfig{
			UserAgent:      "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/104.0.5112.102 Safari/537.36",
			Browser:        "Chrome",
			BrowserVersion: "104.0.5112.102",
			Os:             "Linux",
		},
		Theme: ThemeConfig{
			Background: "black",
			Border:     "white",
			Title:      "white",
		},
		Keys: KeysConfig{
			ToggleGuildsTree:    "Rune[g]",
			ToggleChannelsTree:  "Rune[c]",
			ToggleMessagesPanel: "Rune[m]",
			ToggleMessageInput:  "Rune[i]",

			OpenMessageActionsList: "Rune[a]",
			OpenExternalEditor:     "Ctrl+E",

			SelectPreviousMessage: "Up",
			SelectNextMessage:     "Down",
			SelectFirstMessage:    "Home",
			SelectLastMessage:     "End",
		},
	}
}

func (c *Config) Load(path string) error {
	// Create the configuration directory if it does not exist already, recursively.
	err := os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	// Create a new configuration file if it does not exist already; otherwise, open the existing file with read-write flag.
	f, err := os.OpenFile(filepath.Join(path, "config.js"), os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	// If the configuration file is empty, that is, its size is zero, write the default configuration to the file.
	if fi.Size() == 0 {
	}

	return err
}

func DefaultPath() string {
	path, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	return filepath.Join(path, "discordo")
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
