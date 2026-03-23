package config

import (
	"errors"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/list"
	"github.com/gdamore/tcell/v3"
)

var errInvalidType = errors.New("invalid type")

type Alignment struct{ tview.Alignment }

func (a *Alignment) UnmarshalTOML(v any) error {
	s, ok := v.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "left":
		a.Alignment = tview.AlignmentLeft
	case "center":
		a.Alignment = tview.AlignmentCenter
	case "right":
		a.Alignment = tview.AlignmentRight
	}

	return nil
}

type Style struct{ tcell.Style }

func (s *Style) UnmarshalTOML(value any) error {
	m, ok := value.(map[string]any)
	if !ok {
		return errInvalidType
	}

	// Reset on new styles
	s.Style = tcell.StyleDefault

	for key, val := range m {
		switch key {
		case "foreground":
			if str, ok := val.(string); ok {
				s.Style = s.Foreground(tcell.GetColor(str))
			}
		case "background":
			if str, ok := val.(string); ok {
				s.Style = s.Background(tcell.GetColor(str))
			}
		case "attributes":
			switch val := val.(type) {
			case string:
				s.parseAttr(val)
			case []any:
				for _, attr := range val {
					if str, ok := attr.(string); ok {
						s.parseAttr(str)
					}
				}
			}
		case "underline":
			if str, ok := val.(string); ok {
				switch str {
				case "":
					s.Style = s.Underline(tcell.UnderlineStyleNone)
				case "solid":
					s.Style = s.Underline(tcell.UnderlineStyleSolid)
				case "double":
					s.Style = s.Underline(tcell.UnderlineStyleDouble)
				case "curly":
					s.Style = s.Underline(tcell.UnderlineStyleCurly)
				case "dotted":
					s.Style = s.Underline(tcell.UnderlineStyleDotted)
				case "dashed":
					s.Style = s.Underline(tcell.UnderlineStyleDashed)
				}
			}
		case "underline_color":
			if str, ok := val.(string); ok {
				s.Style = s.Underline(tcell.GetColor(str))
			}
		}
	}

	return nil
}

func (sw *Style) parseAttr(s string) {
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

type BorderSet struct{ tview.BorderSet }

func (bs *BorderSet) UnmarshalTOML(val any) error {
	s, ok := val.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "hidden":
		bs.BorderSet = tview.BorderSetHidden()
	case "plain":
		bs.BorderSet = tview.BorderSetPlain()
	case "round":
		bs.BorderSet = tview.BorderSetRound()
	case "thick":
		bs.BorderSet = tview.BorderSetThick()
	case "double":
		bs.BorderSet = tview.BorderSetDouble()
	}

	return nil
}

type GlyphSet struct{ tview.GlyphSet }

func (gs *GlyphSet) UnmarshalTOML(val any) error {
	s, ok := val.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "minimal":
		gs.GlyphSet = tview.MinimalGlyphSet()
	case "box_drawing", "boxdrawing", "box":
		gs.GlyphSet = tview.BoxDrawingGlyphSet()
	case "unicode":
		gs.GlyphSet = tview.UnicodeGlyphSet()
	}

	return nil
}

type ScrollBarVisibility struct{ list.ScrollBarVisibility }

func (sbv *ScrollBarVisibility) UnmarshalTOML(val any) error {
	s, ok := val.(string)
	if !ok {
		return errInvalidType
	}

	switch s {
	case "automatic", "auto":
		sbv.ScrollBarVisibility = list.ScrollBarVisibilityAutomatic
	case "always":
		sbv.ScrollBarVisibility = list.ScrollBarVisibilityAlways
	case "never", "hidden", "off":
		sbv.ScrollBarVisibility = list.ScrollBarVisibilityNever
	}

	return nil
}

type (
	HelpTheme struct {
		ShortKeyStyle  Style `toml:"short_key_style" default:"{ attributes = 'dim' }"`
		ShortDescStyle Style `toml:"short_desc_style" default:"{}"`
		FullKeyStyle   Style `toml:"full_key_style" default:"{ attributes = 'dim' }"`
		FullDescStyle  Style `toml:"full_desc_style" default:"{}"`
	}

	ThemeStyle struct {
		NormalStyle Style `toml:"normal_style" default:"{ attributes = 'dim' }"`
		ActiveStyle Style `toml:"active_style" default:"{ foreground = 'green', attributes = 'bold' }"`
	}

	TitleTheme struct {
		ThemeStyle

		Alignment Alignment `toml:"alignment" default:"left"`
	}

	FooterTheme struct {
		ThemeStyle

		Alignment Alignment `toml:"alignment" default:"left"`
	}

	BorderTheme struct {
		ThemeStyle

		Enabled bool `toml:"enabled" default:"true"`
		// Border padding order: [top, right, bottom, left].
		Padding [4]int `toml:"padding" default:"[0, 0, 1, 1]"`

		NormalSet BorderSet `toml:"normal_set" default:"round"`
		ActiveSet BorderSet `toml:"active_set" default:"round"`
	}

	GuildsTreeTheme struct {
		AutoExpandFolders bool              `toml:"auto_expand_folders" default:"true"`
		Graphics          bool              `toml:"graphics" default:"true"`
		GraphicsColor     string            `toml:"graphics_color" default:"default"`
		Indents           GuildsTreeIndents `toml:"indents"`
	}

	GuildsTreeIndents struct {
		Guild    int `toml:"guild" default:"2"`
		Category int `toml:"category" default:"1"`
		Channel  int `toml:"channel" default:"2"`
		Forum    int `toml:"forum" default:"2"`
		GroupDM  int `toml:"group_dm" default:"1"`
		DM       int `toml:"dm" default:"2"`
	}

	MessagesListTheme struct {
		ReplyIndicator     string `toml:"reply_indicator" default:">"`
		ForwardedIndicator string `toml:"forwarded_indicator" default:"<"`
		AuthorStyle        Style  `toml:"author_style" default:"{}"`
		MentionStyle       Style  `toml:"mention_style" default:"{ foreground = 'blue', attributes = 'bold' }"`
		EmojiStyle         Style  `toml:"emoji_style" default:"{ foreground = 'green' }"`
		URLStyle           Style  `toml:"url_style" default:"{ foreground = 'blue' }"`
		AttachmentStyle    Style  `toml:"attachment_style" default:"{ foreground = 'yellow' }"`

		MessageStyle         Style `toml:"message_style" default:"{}"`
		SelectedMessageStyle Style `toml:"selected_message_style" default:"{ attributes = 'reverse' }"`

		Embeds MessagesListEmbedsTheme `toml:"embeds"`
	}

	MessagesListEmbedsTheme struct {
		ProviderStyle    Style `toml:"provider_style" default:"{ attributes = ['dim', 'italic'] }"`
		AuthorStyle      Style `toml:"author_style" default:"{ attributes = 'italic' }"`
		TitleStyle       Style `toml:"title_style" default:"{ foreground = 'blue', attributes = 'bold' }"`
		DescriptionStyle Style `toml:"description_style" default:"{ attributes = 'dim' }"`
		FieldNameStyle   Style `toml:"field_name_style" default:"{ attributes = ['bold', 'underline'] }"`
		FieldValueStyle  Style `toml:"field_value_style" default:"{}"`
		FooterStyle      Style `toml:"footer_style" default:"{ attributes = ['dim', 'italic'] }"`
		URLStyle         Style `toml:"url_style" default:"{ foreground = 'blue', underline = 'solid' }"`
	}

	MentionsListTheme struct {
		MinWidth  uint `toml:"min_width" default:"20"`
		MaxHeight uint `toml:"max_height" default:"0"`
	}

	DialogTheme struct {
		Style           Style `toml:"style" default:"{}"`
		BackgroundStyle Style `toml:"background_style" default:"{ attributes = 'dim' }"`
	}

	ScrollBarTheme struct {
		Visibility ScrollBarVisibility `toml:"visibility" default:"auto"`
		GlyphSet   GlyphSet            `toml:"glyph_set" default:"unicode"`
		TrackStyle Style               `toml:"track_style" default:"{ attributes = 'dim' }"`
		ThumbStyle Style               `toml:"thumb_style" default:"{}"`
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
