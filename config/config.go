package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type IdentifyConfig struct {
	UserAgent string `yaml:"user_agent"`
	Os        string `yaml:"os"`
	Browser   string `yaml:"browser"`
}

type GuildsListThemeConfig struct {
	ItemForeground         string `yaml:"item_foreground"`
	SelectedItemForeground string `yaml:"selected_item_foreground"`
}

type MessageInputFieldThemeConfig struct {
	FieldForeground       string `yaml:"field_foreground"`
	PlaceholderForeground string `yaml:"placeholder_foreground"`
}

type ThemeConfig struct {
	Background        string                       `yaml:"background"`
	BorderForeground  string                       `yaml:"border_foreground"`
	TitleForeground   string                       `yaml:"title_foreground"`
	GuildsList        GuildsListThemeConfig        `yaml:"guilds_list"`
	MessageInputField MessageInputFieldThemeConfig `yaml:"message_inputfield"`
}

type Config struct {
	Mouse         bool           `yaml:"mouse"`
	Timestamps    bool           `yaml:"timestamps"`
	MessagesLimit int            `yaml:"messages_limit"`
	Identify      IdentifyConfig `yaml:"identify"`
	Theme         ThemeConfig    `yaml:"theme"`
}

func New() *Config {
	return &Config{
		Mouse:         true,
		Timestamps:    false,
		MessagesLimit: 50,
		Identify: IdentifyConfig{
			UserAgent: userAgent,
			Os:        oss,
			Browser:   browser,
		},
		Theme: ThemeConfig{
			Background:       "default",
			BorderForeground: "white",
			TitleForeground:  "white",
			GuildsList: GuildsListThemeConfig{
				ItemForeground:         "white",
				SelectedItemForeground: "#96CDFB",
			},
			MessageInputField: MessageInputFieldThemeConfig{
				FieldForeground:       "white",
				PlaceholderForeground: "#F2CDCD",
			},
		},
	}
}

func (c *Config) Load() error {
	path, err := os.UserConfigDir()
	if err != nil {
		return err
	}

	// Create all directories mentioned in the path, recursively.
	path = filepath.Join(path, Name)
	err = os.MkdirAll(path, os.ModePerm)
	if err != nil {
		return err
	}

	// If the configuration file does not exist already, a new file is created; otherwise, open an existing file at the path with read-write flag.
	path = filepath.Join(path, "config.yaml")
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		return err
	}

	// If the file is empty, that is, the size of the file is zero, write the default configuration to the newly-created or empty file.
	if fi.Size() == 0 {
		e := yaml.NewEncoder(f)
		e.SetIndent(2)
		return e.Encode(c)
	}

	return yaml.NewDecoder(f).Decode(&c)
}
