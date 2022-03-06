package config

type ThemeConfig struct {
	Background string `toml:"background"`
	Border     string `toml:"border"`
	Title      string `toml:"title"`
}

func newThemeConfig() ThemeConfig {
	return ThemeConfig{
		Background: "black",
		Border:     "white",
		Title:      "white",
	}
}
