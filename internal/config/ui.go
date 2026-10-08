package config

import (
	"fmt"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/scrollbar"
	"github.com/ayn2op/tview/viewport"
)

// enum pairs the names an option accepts with their values; a value is written back as the first of its names.
type enum[T comparable] []struct {
	name  string
	value T
}

func (e enum[T]) parse(name string) (T, error) {
	for _, p := range e {
		if p.name == name {
			return p.value, nil
		}
	}
	var zero T
	return zero, fmt.Errorf("unknown value: %q", name)
}

func (e enum[T]) text(value T) ([]byte, error) {
	for _, p := range e {
		if p.value == value {
			return []byte(p.name), nil
		}
	}
	return nil, fmt.Errorf("unknown value: %v", value)
}

var (
	boxDrawing = scrollbar.SymbolSet{Track: "│", Thumb: "█", Begin: "▲", End: "▼"}

	alignments = enum[tview.Alignment]{
		{"left", tview.AlignmentLeft},
		{"center", tview.AlignmentCenter},
		{"right", tview.AlignmentRight},
	}
	borderSets = enum[tview.BorderSet]{
		{"hidden", tview.BorderSetHidden()},
		{"plain", tview.BorderSetPlain()},
		{"round", tview.BorderSetRound()},
		{"thick", tview.BorderSetThick()},
		{"double", tview.BorderSetDouble()},
	}
	symbolSets = enum[scrollbar.SymbolSet]{
		{"minimal", scrollbar.SymbolSet{Thumb: "█", Begin: "▲", End: "▼"}},
		{"box_drawing", boxDrawing},
		{"boxdrawing", boxDrawing},
		{"box", boxDrawing},
		{"vertical", scrollbar.SymbolSetVertical()},
		{"double_vertical", scrollbar.SymbolSetDoubleVertical()},
	}
	scrollBarVisibilities = enum[viewport.ScrollBarVisibility]{
		{"auto", viewport.ScrollBarVisibilityAutomatic},
		{"automatic", viewport.ScrollBarVisibilityAutomatic},
		{"always", viewport.ScrollBarVisibilityAlways},
		{"never", viewport.ScrollBarVisibilityNever},
		{"hidden", viewport.ScrollBarVisibilityNever},
		{"off", viewport.ScrollBarVisibilityNever},
	}
)

type AlignmentWrapper struct{ tview.Alignment }

func (aw *AlignmentWrapper) UnmarshalText(text []byte) (err error) {
	aw.Alignment, err = alignments.parse(string(text))
	return err
}

func (aw AlignmentWrapper) MarshalText() ([]byte, error) { return alignments.text(aw.Alignment) }

type BorderSetWrapper struct{ tview.BorderSet }

func (bw *BorderSetWrapper) UnmarshalText(text []byte) (err error) {
	bw.BorderSet, err = borderSets.parse(string(text))
	return err
}

func (bw BorderSetWrapper) MarshalText() ([]byte, error) { return borderSets.text(bw.BorderSet) }

type SymbolSetWrapper struct{ scrollbar.SymbolSet }

func (sw *SymbolSetWrapper) UnmarshalText(text []byte) (err error) {
	sw.SymbolSet, err = symbolSets.parse(string(text))
	return err
}

func (sw SymbolSetWrapper) MarshalText() ([]byte, error) { return symbolSets.text(sw.SymbolSet) }

type ScrollBarVisibilityWrapper struct{ viewport.ScrollBarVisibility }

func (vw *ScrollBarVisibilityWrapper) UnmarshalText(text []byte) (err error) {
	vw.ScrollBarVisibility, err = scrollBarVisibilities.parse(string(text))
	return err
}

func (vw ScrollBarVisibilityWrapper) MarshalText() ([]byte, error) {
	return scrollBarVisibilities.text(vw.ScrollBarVisibility)
}

type (
	UIConfig struct {
		// The tabs of the login screen.
		Tabs         TabsUIConfig         `toml:"tabs"`
		Title        TitleUIConfig        `toml:"title"`
		Footer       TitleUIConfig        `toml:"footer"`
		Border       BorderUIConfig       `toml:"border"`
		GuildsTree   GuildsTreeUIConfig   `toml:"guilds_tree"`
		MembersTree  TreeUIConfig         `toml:"members_tree"`
		ScrollBar    ScrollBarUIConfig    `toml:"scroll_bar"`
		MessagesList MessagesListUIConfig `toml:"messages_list"`
		// The list of mention suggestions above the composer. Its width and height are capped to the available space.
		MentionsList MentionsListUIConfig `toml:"mentions_list"`
	}

	TabsUIConfig struct {
		// Where tabs that fit are placed: "left", "center" or "right".
		Alignment AlignmentWrapper `toml:"alignment"`
		// Whether moving past the last tab selects the first, and past the first selects the last.
		Wrap bool `toml:"wrap"`
		// Text between tabs.
		Separator string `toml:"separator"`
		// Text on either side of each tab: [left, right]
		Padding [2]string `toml:"padding"`
		// Arrows shown while tabs are hidden past an end: [start, end]. An empty string turns that arrow off.
		Arrows [2]string `toml:"arrows"`
		// Whether clicking an arrow selects the tab next to the active one.
		ClickableArrows bool `toml:"clickable_arrows"`
	}

	TitleUIConfig struct {
		// "left", "center" or "right".
		Alignment AlignmentWrapper `toml:"alignment"`
	}

	BorderUIConfig struct {
		Enabled bool `toml:"enabled"`
		// [top, bottom, left, right]
		Padding [4]int `toml:"padding"`
		// "hidden", "plain", "round", "thick" or "double".
		NormalSet BorderSetWrapper `toml:"normal_set"`
		// The set used while the widget is focused.
		ActiveSet BorderSetWrapper `toml:"active_set"`
	}

	TreeUIConfig struct {
		// Whether to draw lines that give the tree its shape.
		Graphics bool `toml:"graphics"`
	}

	GuildsTreeUIConfig struct {
		TreeUIConfig
		AutoExpandFolders bool `toml:"auto_expand_folders"`
	}

	ScrollBarUIConfig struct {
		// "auto", "always" or "never".
		Visibility ScrollBarVisibilityWrapper `toml:"visibility"`
		// "minimal", "box_drawing", "vertical" or "double_vertical".
		SymbolSet SymbolSetWrapper `toml:"symbol_set"`
	}

	MessagesListUIConfig struct {
		ReplyIndicator     string `toml:"reply_indicator"`
		ForwardedIndicator string `toml:"forwarded_indicator"`
	}

	MentionsListUIConfig struct {
		// Minimum width. 0 makes the list as wide as possible.
		MinWidth uint `toml:"min_width"`
		// Maximum height. 0 makes the list as tall as needed.
		MaxHeight uint `toml:"max_height"`
	}
)

func defaultUI() UIConfig {
	left := TitleUIConfig{AlignmentWrapper{tview.AlignmentLeft}}
	round := BorderSetWrapper{tview.BorderSetRound()}
	return UIConfig{
		Tabs: TabsUIConfig{
			Alignment:       AlignmentWrapper{tview.AlignmentCenter},
			Separator:       "│",
			Padding:         [2]string{" ", " "},
			Arrows:          [2]string{"◀", "▶"},
			ClickableArrows: true,
		},
		Title:        left,
		Footer:       left,
		Border:       BorderUIConfig{Enabled: true, Padding: [4]int{0, 0, 1, 1}, NormalSet: round, ActiveSet: round},
		GuildsTree:   GuildsTreeUIConfig{TreeUIConfig{Graphics: true}, true},
		MembersTree:  TreeUIConfig{Graphics: true},
		ScrollBar:    ScrollBarUIConfig{SymbolSet: SymbolSetWrapper{boxDrawing}, Visibility: ScrollBarVisibilityWrapper{viewport.ScrollBarVisibilityAutomatic}},
		MessagesList: MessagesListUIConfig{ReplyIndicator: ">", ForwardedIndicator: "<"},
		MentionsList: MentionsListUIConfig{MinWidth: 20},
	}
}
