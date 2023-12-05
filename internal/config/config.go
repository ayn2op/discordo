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

type Keys struct {
	Common map[string]string `yaml:",inline"`

	GuildsTree   map[string]string `yaml:"guilds_tree"`
	MessagesText map[string]string `yaml:"messages_text"`
	MessageInput map[string]string `yaml:"message_input"`
}

type Config struct {
	Mouse                  bool `yaml:"mouse"`
	Timestamps             bool `yaml:"timestamps"`
	TimestampsBeforeAuthor bool `yaml:"timestamps_before_author"`

	MessagesLimit uint8 `yaml:"messages_limit"`

	Editor string `yaml:"editor"`

	Keys struct {
		Normal Keys `yaml:"normal"`
		Insert Keys `yaml:"insert"`

		MessagesText struct {
			ShowImage string `yaml:"show_image"`

			SelectFirst string `yaml:"select_first"`
			SelectLast  string `yaml:"select_last"`
		} `yaml:"messages_text"`
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

	var cfg Config
	if err := yaml.NewDecoder(reader).Decode(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}
