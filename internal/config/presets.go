package config

import (
	"maps"
	"slices"

	"github.com/gdamore/tcell/v3"
)

// pair returns the style with bg as background and fg as foreground.
func pair(bg, fg string) Style {
	return Style{tcell.StyleDefault.Background(tcell.GetColor(bg)).Foreground(tcell.GetColor(fg))}
}

// themes maps the name of each theme to it: the built-in ones, and those Load reads from theme files. The default theme leaves the base unset, so it is the terminal's background.
var themes = map[string]ThemeConfig{
	"default": defaultTheme(),

	"tokyonight":       tokyoNightTheme(),
	"tokyonight_storm": tokyoNightStormTheme(),
	"tokyonight_moon":  tokyoNightMoonTheme(),
	"tokyonight_day":   tokyoNightDayTheme(),

	"catppuccin_mocha":     catppuccinMochaTheme(),
	"catppuccin_macchiato": catppuccinMacchiatoTheme(),
	"catppuccin_frappe":    catppuccinFrappeTheme(),
	"catppuccin_latte":     catppuccinLatteTheme(),

	"dracula":       draculaTheme(),
	"dracula_light": draculaLightTheme(),

	"nord":       nordTheme(),
	"nord_light": nordLightTheme(),

	"gruvbox":       gruvboxTheme(),
	"gruvbox_light": gruvboxLightTheme(),

	"kanagawa":        kanagawaTheme(),
	"kanagawa_dragon": kanagawaDragonTheme(),
	"kanagawa_lotus":  kanagawaLotusTheme(),
}

// Themes returns the names of the themes, sorted.
func Themes() []string {
	return slices.Sorted(maps.Keys(themes))
}

func defaultTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Weak: pair("#3a3d43", "#dcddde"), Neutral: pair("#40444b", "#dcddde"), Strong: pair("#4f545c", "#dcddde"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#7289da", "#dcddde"), Weak: pair("#5b6eae", "#dcddde"), Strong: pair("#8ea1e1", "#2f3136")},
		Secondary: SwatchThemeConfig{Base: pair("#8e9297", "#dcddde"), Weak: pair("#72767d", "#dcddde"), Strong: pair("#b9bbbe", "#2f3136")},
		Success:   pair("#43b581", "#dcddde"),
		Warning:   pair("#d9a95b", "#2f3136"),
		Danger:    pair("#d9686b", "#dcddde"),
	}
}

func tokyoNightTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#1a1b26", "#c0caf5"), Weak: pair("#292e42", "#c0caf5"),
			Neutral: pair("#283457", "#c0caf5"), Strong: pair("#3b4261", "#c0caf5"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#7aa2f7", "#1a1b26"), Weak: pair("#3d59a1", "#c0caf5"), Strong: pair("#7aa2f7", "#1a1b26")},
		Secondary: SwatchThemeConfig{Base: pair("#565f89", "#c0caf5"), Weak: pair("#292e42", "#c0caf5"), Strong: pair("#a9b1d6", "#1a1b26")},
		Success:   pair("#9ece6a", "#1a1b26"),
		Warning:   pair("#e0af68", "#1a1b26"),
		Danger:    pair("#f7768e", "#1a1b26"),
	}
}

func tokyoNightStormTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#24283b", "#c0caf5"), Weak: pair("#292e42", "#c0caf5"),
			Neutral: pair("#2e3c64", "#c0caf5"), Strong: pair("#3b4261", "#c0caf5"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#7aa2f7", "#24283b"), Weak: pair("#3d59a1", "#c0caf5"), Strong: pair("#7aa2f7", "#24283b")},
		Secondary: SwatchThemeConfig{Base: pair("#565f89", "#c0caf5"), Weak: pair("#292e42", "#c0caf5"), Strong: pair("#a9b1d6", "#24283b")},
		Success:   pair("#9ece6a", "#24283b"),
		Warning:   pair("#e0af68", "#24283b"),
		Danger:    pair("#f7768e", "#24283b"),
	}
}

func tokyoNightMoonTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#222436", "#c8d3f5"), Weak: pair("#2f334d", "#c8d3f5"),
			Neutral: pair("#2d3f76", "#c8d3f5"), Strong: pair("#3b4261", "#c8d3f5"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#82aaff", "#222436"), Weak: pair("#3e68d7", "#c8d3f5"), Strong: pair("#82aaff", "#222436")},
		Secondary: SwatchThemeConfig{Base: pair("#636da6", "#c8d3f5"), Weak: pair("#2f334d", "#c8d3f5"), Strong: pair("#828bb8", "#222436")},
		Success:   pair("#c3e88d", "#222436"),
		Warning:   pair("#ffc777", "#222436"),
		Danger:    pair("#ff757f", "#222436"),
	}
}

func tokyoNightDayTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#e1e2e7", "#3760bf"), Weak: pair("#c4c8da", "#3760bf"),
			Neutral: pair("#b7c1e3", "#3760bf"), Strong: pair("#a8aecb", "#3760bf"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#2e7de9", "#e1e2e7"), Weak: pair("#7890dd", "#e1e2e7"), Strong: pair("#2e7de9", "#e1e2e7")},
		Secondary: SwatchThemeConfig{Base: pair("#848cb5", "#e1e2e7"), Weak: pair("#c4c8da", "#3760bf"), Strong: pair("#6172b0", "#e1e2e7")},
		Success:   pair("#587539", "#e1e2e7"),
		Warning:   pair("#8c6c3e", "#e1e2e7"),
		Danger:    pair("#f52a65", "#e1e2e7"),
	}
}

func catppuccinMochaTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#1e1e2e", "#cdd6f4"), Weak: pair("#45475a", "#cdd6f4"),
			Neutral: pair("#45475a", "#cdd6f4"), Strong: pair("#585b70", "#cdd6f4"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#89b4fa", "#1e1e2e"), Weak: pair("#74c7ec", "#1e1e2e"), Strong: pair("#b4befe", "#1e1e2e")},
		Secondary: SwatchThemeConfig{Base: pair("#7f849c", "#cdd6f4"), Weak: pair("#6c7086", "#cdd6f4"), Strong: pair("#a6adc8", "#1e1e2e")},
		Success:   pair("#a6e3a1", "#1e1e2e"),
		Warning:   pair("#f9e2af", "#1e1e2e"),
		Danger:    pair("#f38ba8", "#1e1e2e"),
	}
}

func catppuccinMacchiatoTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#24273a", "#cad3f5"), Weak: pair("#494d64", "#cad3f5"),
			Neutral: pair("#494d64", "#cad3f5"), Strong: pair("#5b6078", "#cad3f5"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#8aadf4", "#24273a"), Weak: pair("#7dc4e4", "#24273a"), Strong: pair("#b7bdf8", "#24273a")},
		Secondary: SwatchThemeConfig{Base: pair("#8087a2", "#24273a"), Weak: pair("#6e738d", "#cad3f5"), Strong: pair("#a5adcb", "#24273a")},
		Success:   pair("#a6da95", "#24273a"),
		Warning:   pair("#eed49f", "#24273a"),
		Danger:    pair("#ed8796", "#24273a"),
	}
}

func catppuccinFrappeTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#303446", "#c6d0f5"), Weak: pair("#51576d", "#c6d0f5"),
			Neutral: pair("#51576d", "#c6d0f5"), Strong: pair("#626880", "#c6d0f5"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#8caaee", "#303446"), Weak: pair("#85c1dc", "#303446"), Strong: pair("#babbf1", "#303446")},
		Secondary: SwatchThemeConfig{Base: pair("#838ba7", "#303446"), Weak: pair("#737994", "#c6d0f5"), Strong: pair("#a5adce", "#303446")},
		Success:   pair("#a6d189", "#303446"),
		Warning:   pair("#e5c890", "#303446"),
		Danger:    pair("#e78284", "#303446"),
	}
}

func catppuccinLatteTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#eff1f5", "#4c4f69"), Weak: pair("#bcc0cc", "#4c4f69"),
			Neutral: pair("#bcc0cc", "#4c4f69"), Strong: pair("#acb0be", "#4c4f69"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#1e66f5", "#eff1f5"), Weak: pair("#209fb5", "#eff1f5"), Strong: pair("#7287fd", "#eff1f5")},
		Secondary: SwatchThemeConfig{Base: pair("#8c8fa1", "#eff1f5"), Weak: pair("#9ca0b0", "#eff1f5"), Strong: pair("#6c6f85", "#eff1f5")},
		Success:   pair("#40a02b", "#eff1f5"),
		Warning:   pair("#df8e1d", "#eff1f5"),
		Danger:    pair("#d20f39", "#eff1f5"),
	}
}

func draculaTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#282a36", "#f8f8f2"), Weak: pair("#424450", "#f8f8f2"),
			Neutral: pair("#44475a", "#f8f8f2"), Strong: pair("#44475a", "#f8f8f2"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#bd93f9", "#282a36"), Weak: pair("#6272a4", "#f8f8f2"), Strong: pair("#bd93f9", "#282a36")},
		Secondary: SwatchThemeConfig{Base: pair("#6272a4", "#f8f8f2"), Weak: pair("#6272a4", "#f8f8f2"), Strong: pair("#f8f8f2", "#282a36")},
		Success:   pair("#50fa7b", "#282a36"),
		Warning:   pair("#ffb86c", "#282a36"),
		Danger:    pair("#ff5555", "#f8f8f2"),
	}
}

func draculaLightTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#fffbeb", "#1f1f1f"), Weak: pair("#dedccf", "#1f1f1f"),
			Neutral: pair("#cfcfde", "#1f1f1f"), Strong: pair("#ceccc0", "#1f1f1f"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#644ac9", "#fffbeb"), Weak: pair("#7862d0", "#fffbeb"), Strong: pair("#644ac9", "#fffbeb")},
		Secondary: SwatchThemeConfig{Base: pair("#6c664b", "#fffbeb"), Weak: pair("#6c664b", "#fffbeb"), Strong: pair("#1f1f1f", "#fffbeb")},
		Success:   pair("#14710a", "#fffbeb"),
		Warning:   pair("#a34d14", "#fffbeb"),
		Danger:    pair("#cb3a2a", "#fffbeb"),
	}
}

func nordTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#2e3440", "#eceff4"), Weak: pair("#3b4252", "#eceff4"),
			Neutral: pair("#434c5e", "#eceff4"), Strong: pair("#4c566a", "#eceff4"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#88c0d0", "#2e3440"), Weak: pair("#5e81ac", "#eceff4"), Strong: pair("#88c0d0", "#2e3440")},
		Secondary: SwatchThemeConfig{Base: pair("#4c566a", "#eceff4"), Weak: pair("#3b4252", "#eceff4"), Strong: pair("#e5e9f0", "#2e3440")},
		Success:   pair("#a3be8c", "#2e3440"),
		Warning:   pair("#ebcb8b", "#2e3440"),
		Danger:    pair("#bf616a", "#eceff4"),
	}
}

func nordLightTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#eceff4", "#2e3440"), Weak: pair("#d8dee9", "#2e3440"),
			Neutral: pair("#e5e9f0", "#2e3440"), Strong: pair("#d8dee9", "#2e3440"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#5e81ac", "#eceff4"), Weak: pair("#81a1c1", "#2e3440"), Strong: pair("#5e81ac", "#eceff4")},
		Secondary: SwatchThemeConfig{Base: pair("#4c566a", "#eceff4"), Weak: pair("#d8dee9", "#2e3440"), Strong: pair("#3b4252", "#eceff4")},
		Success:   pair("#a3be8c", "#2e3440"),
		Warning:   pair("#ebcb8b", "#2e3440"),
		Danger:    pair("#bf616a", "#eceff4"),
	}
}

func gruvboxTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#282828", "#ebdbb2"), Weak: pair("#504945", "#ebdbb2"),
			Neutral: pair("#665c54", "#ebdbb2"), Strong: pair("#665c54", "#ebdbb2"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#83a598", "#282828"), Weak: pair("#458588", "#ebdbb2"), Strong: pair("#8ec07c", "#282828")},
		Secondary: SwatchThemeConfig{Base: pair("#928374", "#282828"), Weak: pair("#665c54", "#ebdbb2"), Strong: pair("#a89984", "#282828")},
		Success:   pair("#b8bb26", "#282828"),
		Warning:   pair("#fabd2f", "#282828"),
		Danger:    pair("#fb4934", "#ebdbb2"),
	}
}

func gruvboxLightTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#fbf1c7", "#3c3836"), Weak: pair("#d5c4a1", "#3c3836"),
			Neutral: pair("#bdae93", "#3c3836"), Strong: pair("#bdae93", "#3c3836"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#076678", "#fbf1c7"), Weak: pair("#458588", "#fbf1c7"), Strong: pair("#427b58", "#fbf1c7")},
		Secondary: SwatchThemeConfig{Base: pair("#928374", "#fbf1c7"), Weak: pair("#bdae93", "#3c3836"), Strong: pair("#7c6f64", "#fbf1c7")},
		Success:   pair("#79740e", "#fbf1c7"),
		Warning:   pair("#b57614", "#fbf1c7"),
		Danger:    pair("#9d0006", "#fbf1c7"),
	}
}

func kanagawaTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#1f1f28", "#dcd7ba"), Weak: pair("#363646", "#dcd7ba"),
			Neutral: pair("#223249", "#dcd7ba"), Strong: pair("#54546d", "#dcd7ba"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#7e9cd8", "#1f1f28"), Weak: pair("#2d4f67", "#dcd7ba"), Strong: pair("#7fb4ca", "#1f1f28")},
		Secondary: SwatchThemeConfig{Base: pair("#727169", "#dcd7ba"), Weak: pair("#54546d", "#dcd7ba"), Strong: pair("#c8c093", "#1f1f28")},
		Success:   pair("#76946a", "#1f1f28"),
		Warning:   pair("#dca561", "#1f1f28"),
		Danger:    pair("#c34043", "#dcd7ba"),
	}
}

func kanagawaDragonTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#181616", "#c5c9c5"), Weak: pair("#393836", "#c5c9c5"),
			Neutral: pair("#223249", "#c5c9c5"), Strong: pair("#625e5a", "#c5c9c5"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#8ba4b0", "#181616"), Weak: pair("#2d4f67", "#c5c9c5"), Strong: pair("#949fb5", "#181616")},
		Secondary: SwatchThemeConfig{Base: pair("#737c73", "#181616"), Weak: pair("#625e5a", "#c5c9c5"), Strong: pair("#c8c093", "#181616")},
		Success:   pair("#76946a", "#181616"),
		Warning:   pair("#dca561", "#181616"),
		Danger:    pair("#c34043", "#c5c9c5"),
	}
}

func kanagawaLotusTheme() ThemeConfig {
	return ThemeConfig{
		Background: BackgroundThemeConfig{
			Base: pair("#f2ecbc", "#545464"), Weak: pair("#e4d794", "#545464"),
			Neutral: pair("#c9cbd1", "#545464"), Strong: pair("#a09cac", "#f2ecbc"),
		},
		Primary:   SwatchThemeConfig{Base: pair("#4d699b", "#f2ecbc"), Weak: pair("#b5cbd2", "#545464"), Strong: pair("#6693bf", "#f2ecbc")},
		Secondary: SwatchThemeConfig{Base: pair("#8a8980", "#f2ecbc"), Weak: pair("#a09cac", "#f2ecbc"), Strong: pair("#43436c", "#f2ecbc")},
		Success:   pair("#6e915f", "#f2ecbc"),
		Warning:   pair("#de9800", "#f2ecbc"),
		Danger:    pair("#d7474b", "#f2ecbc"),
	}
}
