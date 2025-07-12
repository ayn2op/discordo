package config

import "github.com/ayn2op/tview"

type (
	TitleTheme struct {
		ThemeStyle
		Alignment AlignmentWrapper `toml:"alignment"`
	}

	BorderTheme struct {
		ThemeStyle
		Enabled bool             `toml:"enabled"`
		Padding [4]int           `toml:"padding"`
		Set     BorderSetWrapper `toml:"set"`
	}
)

type BorderSetWrapper struct{ tview.BorderSet }

func (bw *BorderSetWrapper) UnmarshalTOML(val any) error {
	s, ok := val.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "plain":
		bw.BorderSet = tview.BorderSetPlain()
	case "round":
		bw.BorderSet = tview.BorderSetRound()
	case "thick":
		bw.BorderSet = tview.BorderSetThick()
	case "double":
		bw.BorderSet = tview.BorderSetDouble()
	}

	return nil
}
