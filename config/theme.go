package config

type ThemeConfig struct {
	Background    string `toml:"background"`
	Border        string `toml:"border"`
	Title         string `toml:"title"`
	ChannelUpdate string `toml:"channelUpdate"`
	MentionSelf   string `toml:"mentionSelf"`
	MentionOther  string `toml:"mentionOther"`
	NameSelf      string `toml:"nameSelf"`
	NameOther     string `toml:"nameOther"`
	NameBot       string `toml:"nameBot"`
}

func newThemeConfig() ThemeConfig {
	return ThemeConfig{
		Background:    "black",
		Border:        "white",
		Title:         "white",
		ChannelUpdate: "#5865F2",
		MentionSelf:   "#5865F2",
		MentionOther:  "#EB459E",
		NameSelf:      "#57F287",
		NameOther:     "#ED4245",
		NameBot:       "#EB459E",
	}
}
