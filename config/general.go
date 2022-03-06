package config

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
}

func newGeneralConfig() GeneralConfig {
	return GeneralConfig{
		UserAgent:          "Mozilla/5.0 (X11; Linux x86_64; rv:97.0) Gecko/20100101 Firefox/97.0",
		FetchMessagesLimit: 50,
		Mouse:              true,
		Timestamps:         false,
		Identify: IdentifyConfig{
			Os:      "Linux",
			Browser: "Firefox",
		},
	}
}
