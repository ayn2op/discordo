package config

import (
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/BurntSushi/toml"
)

type IdentifyConfig struct {
	UserAgent string `toml:"user_agent"`
	Os        string `toml:"os"`
	Browser   string `toml:"browser"`
}

type KeysConfig struct {
	ToggleGuildsList        string `toml:"toggle_guilds_list"`
	ToggleChannelsTreeView  string `toml:"toggle_channels_tree_view"`
	ToggleMessagesTextView  string `toml:"toggle_messages_text_view"`
	ToggleMessageInputField string `toml:"toggle_message_input_field"`

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
	MessagesLimit          int            `toml:"messages_limit"`
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
		AttachmentDownloadsDir: UserDownloadsDir(),
		Identify: IdentifyConfig{
			UserAgent: "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/101.0.4951.67 Safari/537.36",
			Os:        "Linux",
			Browser:   "Chrome",
		},
		Theme: ThemeConfig{
			Background: "black",
			Border:     "white",
			Title:      "white",
		},
		Keys: KeysConfig{
			ToggleGuildsList:        "Rune[g]",
			ToggleChannelsTreeView:  "Rune[c]",
			ToggleMessagesTextView:  "Rune[m]",
			ToggleMessageInputField: "Rune[i]",

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
	// Create directories that do not exist and are mentioned in the path recursively.
	err := os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return err
	}

	// If the configuration file does not exist already, create a new file; otherwise, open the existing file with read-write flag.
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	// If the file is empty (the size of the file is zero), write the default configuration to the file.
	if fi.Size() == 0 {
		return toml.NewEncoder(f).Encode(c)
	}

	_, err = toml.NewDecoder(f).Decode(&c)
	return err
}

func DefaultPath() string {
	path, err := os.UserConfigDir()
	if err != nil {
		log.Fatal(err)
	}

	path += "/discordo/config.toml"
	return path
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
