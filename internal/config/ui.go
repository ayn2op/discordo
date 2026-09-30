package config

type (
	UIConfig struct {
		Tabs TabsConfig `toml:"tabs"`
	}

	TabsConfig struct {
		Alignment       AlignmentWrapper `toml:"alignment"`
		Wrap            bool             `toml:"wrap"`
		Separator       string           `toml:"separator"`
		Padding         [2]string        `toml:"padding"`
		Arrows          [2]string        `toml:"arrows"`
		ClickableArrows bool             `toml:"clickable_arrows"`
	}
)
