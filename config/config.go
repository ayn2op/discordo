package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/robertkrimen/otto"
)

type IdentifyConfig struct {
	UserAgent      string
	Browser        string
	BrowserVersion string
	Os             string
}

type KeysConfig struct {
	ToggleGuildsTree    string
	ToggleChannelsTree  string
	ToggleMessagesPanel string
	ToggleMessageInput  string

	OpenMessageActionsList string
	OpenExternalEditor     string

	SelectPreviousMessage string
	SelectNextMessage     string
	SelectFirstMessage    string
	SelectLastMessage     string
}

type ThemeConfig struct {
	Background string
	Border     string
	Title      string
}

type Config struct {
	Mouse                  bool
	Timestamps             bool
	MessagesLimit          uint
	Timezone               string
	TimeFormat             string
	AttachmentDownloadsDir string
	Identify               IdentifyConfig
	Theme                  ThemeConfig
	Keys                   KeysConfig

	VM *otto.Otto
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

		VM: otto.New(),
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
