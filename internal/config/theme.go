package config

import (
	"errors"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

var errInvalidType = errors.New("invalid type")

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
	}

	return nil
}

type StyleWrapper struct{ tcell.Style }

func NewStyleWrapper(style tcell.Style) StyleWrapper {
	return StyleWrapper{Style: style}
}

func (sw *StyleWrapper) UnmarshalTOML(v any) error {
	m, ok := v.(map[string]any)
	if !ok {
		return errInvalidType
	}

	// Reset on new styles
	sw.Style = tcell.StyleDefault

	for key, val := range m {
		switch key {
		case "foreground":
			if s, ok := val.(string); ok {
				sw.Style = sw.Foreground(tcell.GetColor(s))
			}
		case "background":
			if s, ok := val.(string); ok {
				sw.Style = sw.Background(tcell.GetColor(s))
			}
		case "attributes":
			switch val := val.(type) {
			case string:
				sw.parseAttr(val)
			case []any:
				for _, attr := range val {
					if s, ok := attr.(string); ok {
						sw.parseAttr(s)
					}
				}

			}
		case "underline":
			if s, ok := val.(string); ok {
				switch s {
				case "": sw.Style = sw.Underline(tcell.UnderlineStyleNone)
				case "solid": sw.Style = sw.Underline(tcell.UnderlineStyleSolid)
				case "double": sw.Style = sw.Underline(tcell.UnderlineStyleDouble)
				case "curly": sw.Style = sw.Underline(tcell.UnderlineStyleCurly)
				case "dotted": sw.Style = sw.Underline(tcell.UnderlineStyleDotted)
				case "dashed": sw.Style = sw.Underline(tcell.UnderlineStyleDashed)
				}
			}
		case "underline_color":
			if s, ok := val.(string); ok {
				sw.Style = sw.Underline(tcell.GetColor(s))
			}
		}
	}

	return nil
}

func (sw *StyleWrapper) parseAttr(s string) {
	switch s {
	case "underline":
		sw.Style = sw.Underline(true)
	case "bold":
		sw.Style = sw.Bold(true)
	case "blink":
		sw.Style = sw.Blink(true)
	case "reverse":
		sw.Style = sw.Reverse(true)
	case "dim":
		sw.Style = sw.Dim(true)
	case "italic":
		sw.Style = sw.Italic(true)
	case "strikethrough":
		sw.Style = sw.StrikeThrough(true)
	}
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
	}

	return nil
}

type GlyphSetWrapper struct{ tview.GlyphSet }

func (gw *GlyphSetWrapper) UnmarshalTOML(val any) error {
	s, ok := val.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "minimal":
		gw.GlyphSet = tview.MinimalGlyphSet()
	case "box_drawing", "boxdrawing", "box":
		gw.GlyphSet = tview.BoxDrawingGlyphSet()
	case "unicode":
		gw.GlyphSet = tview.UnicodeGlyphSet()
	}

	return nil
}

type ScrollBarVisibilityWrapper struct{ tview.ScrollBarVisibility }

func (vw *ScrollBarVisibilityWrapper) UnmarshalTOML(val any) error {
	s, ok := val.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "automatic", "auto":
		vw.ScrollBarVisibility = tview.ScrollBarVisibilityAutomatic
	case "always":
		vw.ScrollBarVisibility = tview.ScrollBarVisibilityAlways
	case "never", "hidden", "off":
		vw.ScrollBarVisibility = tview.ScrollBarVisibilityNever
	}

	return nil
}

type HelpTheme struct {
	ShortKeyStyle  StyleWrapper `toml:"short_key_style"`
	ShortDescStyle StyleWrapper `toml:"short_desc_style"`
	FullKeyStyle   StyleWrapper `toml:"full_key_style"`
	FullDescStyle  StyleWrapper `toml:"full_desc_style"`
}

type (
	ThemeStyle struct {
		NormalStyle StyleWrapper `toml:"normal_style"`
		ActiveStyle StyleWrapper `toml:"active_style"`
	}

	TitleTheme struct {
		ThemeStyle
		Alignment AlignmentWrapper `toml:"alignment"`
	}

	FooterTheme struct {
		ThemeStyle
		Alignment AlignmentWrapper `toml:"alignment"`
	}

	BorderTheme struct {
		ThemeStyle
		Enabled bool   `toml:"enabled"`
		Padding [4]int `toml:"padding"`

		NormalSet BorderSetWrapper `toml:"normal_set"`
		ActiveSet BorderSetWrapper `toml:"active_set"`
	}

	GuildsTreeTheme struct {
		AutoExpandFolders bool              `toml:"auto_expand_folders"`
		Graphics          bool              `toml:"graphics"`
		GraphicsColor     string            `toml:"graphics_color"`
		Indents           GuildsTreeIndents `toml:"indents"`
	}

	GuildsTreeIndents struct {
		Guild    int `toml:"guild"`
		Category int `toml:"category"`
		Channel  int `toml:"channel"`
		GroupDM  int `toml:"groupdm"`
		DM       int `toml:"dm"`
	}

	MessagesListTheme struct {
		ReplyIndicator     string       `toml:"reply_indicator"`
		ForwardedIndicator string       `toml:"forwarded_indicator"`
		AuthorStyle        StyleWrapper `toml:"author_style"`
		MentionStyle       StyleWrapper `toml:"mention_style"`
		EmojiStyle         StyleWrapper `toml:"emoji_style"`
		URLStyle           StyleWrapper `toml:"url_style"`
		AttachmentStyle    StyleWrapper `toml:"attachment_style"`

		MessageStyle         StyleWrapper `toml:"message_style"`
		SelectedMessageStyle StyleWrapper `toml:"selected_message_style"`
	}

	MentionsListTheme struct {
		MinWidth  uint `toml:"min_width"`
		MaxHeight uint `toml:"max_height"`
	}

	DialogTheme struct {
		Style           StyleWrapper `toml:"style"`
		BackgroundStyle StyleWrapper `toml:"background_style"`
	}

	ScrollBarTheme struct {
		Visibility ScrollBarVisibilityWrapper `toml:"visibility"`
		GlyphSet   GlyphSetWrapper            `toml:"glyph_set"`
		TrackStyle StyleWrapper               `toml:"track_style"`
		ThumbStyle StyleWrapper               `toml:"thumb_style"`
	}

	Theme struct {
		Title        TitleTheme        `toml:"title"`
		Footer       FooterTheme       `toml:"footer"`
		Border       BorderTheme       `toml:"border"`
		GuildsTree   GuildsTreeTheme   `toml:"guilds_tree"`
		ScrollBar    ScrollBarTheme    `toml:"scroll_bar"`
		MessagesList MessagesListTheme `toml:"messages_list"`
		MentionsList MentionsListTheme `toml:"mentions_list"`
		Dialog       DialogTheme       `toml:"dialog"`
		Help         HelpTheme         `toml:"help"`
	}
)
