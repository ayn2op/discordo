package config

type (
	Notifications struct {
		Enabled   bool   `toml:"enabled"`
		Duration  int    `toml:"duration"`
		PlayChime Chimes `toml:"play_chime"`
	}

	Chimes struct {
		Enabled    bool `toml:"enabled"`
		OnlyOnPing bool `toml:"only_on_ping"`
	}
)

func defaultNotifications() Notifications {
	return Notifications{
		Enabled:  true,
		Duration: 500,
		PlayChime: Chimes{
			Enabled:    true,
			OnlyOnPing: true,
		},
	}
}
