package config

type (
	Notifications struct {
		Enabled  bool   `toml:"enabled"`
		Duration int    `toml:"duration"`
		Sounds   Sounds `toml:"sounds"`
	}

	Sounds struct {
		Enabled    bool `toml:"enabled"`
		OnlyOnPing bool `toml:"only_on_ping"`
	}
)

func defaultNotifications() Notifications {
	return Notifications{
		Enabled:  true,
		Duration: 500,
		Sounds: Sounds{
			Enabled:    true,
			OnlyOnPing: true,
		},
	}
}
