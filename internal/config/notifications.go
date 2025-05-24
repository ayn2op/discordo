package config

type (
	Notifications struct {
		Enabled  bool  `toml:"enabled"`
		Duration int   `toml:"duration"`
		Sound    Sound `toml:"sound"`
	}

	Sound struct {
		Enabled    bool `toml:"enabled"`
		OnlyOnPing bool `toml:"only_on_ping"`
	}
)

func defaultNotifications() Notifications {
	return Notifications{
		Enabled:  true,
		Duration: 500,
		Sound: Sound{
			Enabled:    true,
			OnlyOnPing: true,
		},
	}
}
