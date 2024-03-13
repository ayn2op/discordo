package config

import (
	"bytes"
	_ "embed"
	"io"
	"os"
	"path/filepath"

	"github.com/ayn2op/discordo/internal/constants"
	"gopkg.in/yaml.v3"
)

//go:embed config.yml
var defaultConfig []byte

type Config struct {
	Mouse bool `yaml:"mouse"`

	Timestamps             bool `yaml:"timestamps"`
	TimestampsBeforeAuthor bool `yaml:"timestamps_before_author"`

	MessagesLimit uint8 `yaml:"messages_limit"`

	Editor string `yaml:"editor"`

	Keys struct {
		Cancel string `yaml:"cancel"`

		GuildsTree struct {
			Focus  string `yaml:"focus"`
			Toggle string `yaml:"toggle"`
		} `yaml:"guilds_tree"`

		MessagesText struct {
			Focus string `yaml:"focus"`

			ShowImage   string `yaml:"show_image"`
			CopyContent string `yaml:"copy_content"`

			Reply        string `yaml:"reply"`
			ReplyMention string `yaml:"reply_mention"`

			Delete string `yaml:"delete"`

			SelectPrevious string `yaml:"select_previous"`
			SelectNext     string `yaml:"select_next"`
			SelectFirst    string `yaml:"select_first"`
			SelectLast     string `yaml:"select_last"`
			SelectReply    string `yaml:"select_reply"`
		} `yaml:"messages_text"`

		MessageInput struct {
			Focus string `yaml:"focus"`

			Send         string `yaml:"send"`
			LaunchEditor string `yaml:"launch_editor"`
		} `yaml:"message_input"`
	} `yaml:"keys"`
	Theme struct {
		Border        bool   `yaml:"border"`
		BorderColor   string `yaml:"border_color"`
		BorderPadding [4]int `yaml:"border_padding,flow"`

		TitleColor      string `yaml:"title_color"`
		BackgroundColor string `yaml:"background_color"`

		GuildsTree struct {
			Graphics bool `yaml:"graphics"`
		} `yaml:"guilds_tree"`

		MessagesText struct {
			AuthorColor    string `yaml:"author_color"`
			ReplyIndicator string `yaml:"reply_indicator"`
		} `yaml:"messages_text"`
	} `yaml:"theme"`
}

// Recursively creates the configuration directory if it does not exist already and returns the path to the configuration file.
func initialize() (string, error) {
	path, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}

	path = filepath.Join(path, constants.Name)
	if err := os.MkdirAll(path, os.ModePerm); err != nil {
		return "", err
	}

	return filepath.Join(path, "config.yml"), nil
}

// Reads the configuration file and parses it.
func Load() (*Config, error) {
	path, err := initialize()
	if err != nil {
		return nil, err
	}

	f, err := os.Open(path)
	reader := io.Reader(f)
	if os.IsNotExist(err) {
		err = os.WriteFile(path, defaultConfig, os.ModePerm)
		reader = bytes.NewReader(defaultConfig)
	}

	if err != nil {
		return nil, err
	}
	defer f.Close()

	// Initialize config struct with default values
	var cfg Config
	cfg.Mouse                             = true
	cfg.Timestamps                        = false
	cfg.TimestampsBeforeAuthor            = false
	cfg.MessagesLimit                     = 50
	cfg.Editor                            = "default"

	cfg.Keys.Cancel                       = "Esc"

	cfg.Keys.GuildsTree.Focus             = "Ctrl+G"
	cfg.Keys.GuildsTree.Toggle            = "Ctrl+B"

	cfg.Keys.MessagesText.Focus           = "Ctrl+T"
	cfg.Keys.MessagesText.ShowImage       = "i"
	cfg.Keys.MessagesText.CopyContent     = "c"
	cfg.Keys.MessagesText.Delete          = "d"
	cfg.Keys.MessagesText.Reply           = "r"
	cfg.Keys.MessagesText.ReplyMention    = "R"
	cfg.Keys.MessagesText.SelectPrevious  = "Up"
	cfg.Keys.MessagesText.SelectNext      = "Down"
	cfg.Keys.MessagesText.SelectFirst     = "Home"
	cfg.Keys.MessagesText.SelectLast      = "End"
	cfg.Keys.MessagesText.SelectReply     = "s"

	cfg.Keys.MessageInput.Focus           = "Ctrl+P"
	cfg.Keys.MessageInput.Send            = "Enter"
	cfg.Keys.MessageInput.LaunchEditor    = "Ctrl+E"

	cfg.Theme.Border                      = true
	cfg.Theme.BorderColor                 = "default"
	cfg.Theme.BorderPadding               = [4]int{0, 0, 1, 1}
	cfg.Theme.TitleColor                  = "default"
	cfg.Theme.BackgroundColor             = "default"

	cfg.Theme.GuildsTree.Graphics         = true

	cfg.Theme.MessagesText.AuthorColor    = "aqua"
	cfg.Theme.MessagesText.ReplyIndicator = "╭"

	// Overwrite default values via config file
	decoder := yaml.NewDecoder(reader)
	if err := decoder.Decode(&cfg); err != nil {
		// Decoder might reach end of file without decoding anything (empty file) - this is not an issue
		if err.Error() == "EOF" {
		} else {
			return nil, err
		}
	}

	return &cfg, nil
}
