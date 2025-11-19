package config

import (
	"github.com/ayn2op/tview"
)

type AlignmentWrapper struct{ tview.Alignment }

func (aw *AlignmentWrapper) UnmarshalTOML(value any) error {
	s, ok := value.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "left":
		aw.Alignment = tview.AlignmentLeft
	case "center":
		aw.Alignment = tview.AlignmentCenter
	case "right":
		aw.Alignment = tview.AlignmentRight
	}

	return nil
}

type BorderSetWrapper struct{ tview.BorderSet }

func (bw *BorderSetWrapper) UnmarshalTOML(value any) error {
	s, ok := value.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "hidden":
		bw.BorderSet = tview.BorderSetHidden()
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

type (
	ThemeStyle struct {
		NormalStyle StyleWrapper `toml:"normal_style" default:"attributes:dim"`
		ActiveStyle StyleWrapper `toml:"active_style" default:"foreground:green;attributes:bold"`
	}

	TitleTheme struct {
		ThemeStyle
		Alignment AlignmentWrapper `toml:"alignment" default:"left" description:"Alignment of the title text. Possible values: \"left\", \"center\", \"right\"."`
	}

	BorderTheme struct {
		ThemeStyle
		Enabled bool   `toml:"enabled" default:"true"`
		Padding [4]int `toml:"padding" default:"0 0 1 1" description:"Padding of the border. Order: top, bottom, left, right."`

		NormalSet BorderSetWrapper `toml:"normal_set" default:"round" description:"Normal borders. Possible values: \"hidden\", \"plain\", \"round\", \"thick\", or \"double\"."`
		ActiveSet BorderSetWrapper `toml:"active_set" default:"round" description:"Active borders. Possible values: \"hidden\", \"plain\", \"round\", \"thick\", or \"double\"."`
	}
)
