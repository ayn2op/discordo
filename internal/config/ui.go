package config

import (
	"fmt"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/scrollbar"
)

type AlignmentWrapper struct{ tview.Alignment }

func (aw *AlignmentWrapper) UnmarshalTOML(v any) error {
	s, ok := v.(string)
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
	default:
		return fmt.Errorf("unknown value: %q", s)
	}
	return nil
}

type BorderSetWrapper struct{ tview.BorderSet }

func (bw *BorderSetWrapper) UnmarshalTOML(val any) error {
	s, ok := val.(string)
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
	default:
		return fmt.Errorf("unknown value: %q", s)
	}
	return nil
}

type SymbolSetWrapper struct{ scrollbar.SymbolSet }

func (sw *SymbolSetWrapper) UnmarshalTOML(val any) error {
	s, ok := val.(string)
	if !ok {
		return errInvalidType
	}
	switch s {
	case "minimal":
		sw.SymbolSet = scrollbar.SymbolSet{Thumb: "█", Begin: "▲", End: "▼"}
	case "box_drawing", "boxdrawing", "box":
		sw.SymbolSet = scrollbar.SymbolSet{Track: "│", Thumb: "█", Begin: "▲", End: "▼"}
	case "vertical":
		sw.SymbolSet = scrollbar.SymbolSetVertical()
	case "double_vertical":
		sw.SymbolSet = scrollbar.SymbolSetDoubleVertical()
	default:
		return fmt.Errorf("unknown value: %q", s)
	}
	return nil
}

type ScrollBarVisibilityWrapper struct{ list.ScrollBarVisibility }

func (vw *ScrollBarVisibilityWrapper) UnmarshalTOML(val any) error {
	s, ok := val.(string)
	if !ok {
		return errInvalidType
	}
	switch s {
	case "automatic", "auto":
		vw.ScrollBarVisibility = list.ScrollBarVisibilityAutomatic
	case "always":
		vw.ScrollBarVisibility = list.ScrollBarVisibilityAlways
	case "never", "hidden", "off":
		vw.ScrollBarVisibility = list.ScrollBarVisibilityNever
	default:
		return fmt.Errorf("unknown value: %q", s)
	}
	return nil
}

type (
	UIConfig struct {
		Tabs         TabsConfig         `toml:"tabs"`
		Title        TitleConfig        `toml:"title"`
		Footer       TitleConfig        `toml:"footer"`
		Border       BorderConfig       `toml:"border"`
		GuildsTree   GuildsTreeConfig   `toml:"guilds_tree"`
		MembersTree  TreeConfig         `toml:"members_tree"`
		ScrollBar    ScrollBarConfig    `toml:"scroll_bar"`
		MessagesList MessagesListConfig `toml:"messages_list"`
		MentionsList MentionsListConfig `toml:"mentions_list"`
	}

	TabsConfig struct {
		Alignment       AlignmentWrapper `toml:"alignment"`
		Wrap            bool             `toml:"wrap"`
		Separator       string           `toml:"separator"`
		Padding         [2]string        `toml:"padding"`
		Arrows          [2]string        `toml:"arrows"`
		ClickableArrows bool             `toml:"clickable_arrows"`
	}

	TitleConfig struct {
		Alignment AlignmentWrapper `toml:"alignment"`
	}

	BorderConfig struct {
		Enabled   bool             `toml:"enabled"`
		Padding   [4]int           `toml:"padding"`
		NormalSet BorderSetWrapper `toml:"normal_set"`
		ActiveSet BorderSetWrapper `toml:"active_set"`
	}

	TreeConfig struct {
		Graphics bool `toml:"graphics"`
	}

	GuildsTreeConfig struct {
		TreeConfig
		AutoExpandFolders bool `toml:"auto_expand_folders"`
	}

	ScrollBarConfig struct {
		Visibility ScrollBarVisibilityWrapper `toml:"visibility"`
		SymbolSet  SymbolSetWrapper           `toml:"symbol_set"`
	}

	MessagesListConfig struct {
		ReplyIndicator     string `toml:"reply_indicator"`
		ForwardedIndicator string `toml:"forwarded_indicator"`
	}

	MentionsListConfig struct {
		MinWidth  uint `toml:"min_width"`
		MaxHeight uint `toml:"max_height"`
	}
)
