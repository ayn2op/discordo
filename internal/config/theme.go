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
		ShortKeyStyle  Style `toml:"short_key_style"`
		ShortDescStyle Style `toml:"short_desc_style"`
		FullKeyStyle   Style `toml:"full_key_style"`
		FullDescStyle  Style `toml:"full_desc_style"`
	}

	ThemeStyle struct {
		NormalStyle Style `toml:"normal_style"`
		ActiveStyle Style `toml:"active_style"`
	}

	TitleTheme struct {
		ThemeStyle
		Alignment Alignment `toml:"alignment"`
	}

	FooterTheme struct {
		ThemeStyle
		Alignment Alignment `toml:"alignment"`
	}

	BorderTheme struct {
		ThemeStyle
		Enabled bool `toml:"enabled"`
		// Border padding order: [top, right, bottom, left].
		Padding [4]int `toml:"padding"`

		NormalSet BorderSet `toml:"normal_set"`
		ActiveSet BorderSet `toml:"active_set"`
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
		Forum    int `toml:"forum"`
		GroupDM  int `toml:"group_dm"`
		DM       int `toml:"dm"`
	}

	MessagesListTheme struct {
		ReplyIndicator     string `toml:"reply_indicator"`
		ForwardedIndicator string `toml:"forwarded_indicator"`
		AuthorStyle        Style  `toml:"author_style"`
		MentionStyle       Style  `toml:"mention_style"`
		EmojiStyle         Style  `toml:"emoji_style"`
		URLStyle           Style  `toml:"url_style"`
		AttachmentStyle    Style  `toml:"attachment_style"`

		MessageStyle         Style `toml:"message_style"`
		SelectedMessageStyle Style `toml:"selected_message_style"`

		Embeds MessagesListEmbedsTheme `toml:"embeds"`
	}

	MessagesListEmbedsTheme struct {
		ProviderStyle    Style `toml:"provider_style"`
		AuthorStyle      Style `toml:"author_style"`
		TitleStyle       Style `toml:"title_style"`
		DescriptionStyle Style `toml:"description_style"`
		FieldNameStyle   Style `toml:"field_name_style"`
		FieldValueStyle  Style `toml:"field_value_style"`
		FooterStyle      Style `toml:"footer_style"`
		URLStyle         Style `toml:"url_style"`
	}

	MentionsListTheme struct {
		MinWidth  uint `toml:"min_width"`
		MaxHeight uint `toml:"max_height"`
	}

	DialogTheme struct {
		Style           Style `toml:"style"`
		BackgroundStyle Style `toml:"background_style"`
	}

	ScrollBarTheme struct {
		Visibility ScrollBarVisibility `toml:"visibility"`
		GlyphSet   GlyphSet            `toml:"glyph_set"`
		TrackStyle Style               `toml:"track_style"`
		ThumbStyle Style               `toml:"thumb_style"`
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

func defaultTheme() Theme {
	return Theme{
		Title: TitleTheme{
			ThemeStyle: ThemeStyle{
				NormalStyle: styleAttrs("dim"),
				ActiveStyle: styleColorAttrs("green", "bold"),
			},
			Alignment: Alignment{Alignment: tview.AlignmentLeft},
		},
		Footer: FooterTheme{
			ThemeStyle: ThemeStyle{
				NormalStyle: styleAttrs("dim"),
				ActiveStyle: styleColorAttrs("green", "bold"),
			},
			Alignment: Alignment{Alignment: tview.AlignmentLeft},
		},
		Border: BorderTheme{
			ThemeStyle: ThemeStyle{
				NormalStyle: styleAttrs("dim"),
				ActiveStyle: styleColorAttrs("green", "bold"),
			},
			Enabled:   true,
			Padding:   [4]int{0, 0, 1, 1},
			NormalSet: BorderSet{BorderSet: tview.BorderSetRound()},
			ActiveSet: BorderSet{BorderSet: tview.BorderSetRound()},
		},
		GuildsTree: GuildsTreeTheme{
			AutoExpandFolders: true,
			Graphics:          true,
			GraphicsColor:     "default",
			Indents: GuildsTreeIndents{
				DM:       2,
				GroupDM:  1,
				Guild:    2,
				Category: 1,
				Channel:  2,
				Forum:    2,
			},
		},
		ScrollBar: ScrollBarTheme{
			Visibility: ScrollBarVisibility{ScrollBarVisibility: list.ScrollBarVisibilityAutomatic},
			GlyphSet:   GlyphSet{GlyphSet: tview.UnicodeGlyphSet()},
			TrackStyle: styleAttrs("dim"),
			ThumbStyle: defaultStyle(),
		},
		MessagesList: MessagesListTheme{
			ReplyIndicator:     ">",
			ForwardedIndicator: "<",
			AuthorStyle:        defaultStyle(),
			MentionStyle:       styleColorAttrs("blue", "bold"),
			EmojiStyle:         styleColor("green"),
			URLStyle:           styleColor("blue"),
			AttachmentStyle:    styleColor("yellow"),
			MessageStyle:       defaultStyle(),
			SelectedMessageStyle: Style{
				Style: tcell.StyleDefault.Reverse(true),
			},
			Embeds: MessagesListEmbedsTheme{
				ProviderStyle:    styleAttrs("dim", "italic"),
				AuthorStyle:      styleAttrs("italic"),
				TitleStyle:       styleColorAttrs("blue", "bold"),
				DescriptionStyle: styleAttrs("dim"),
				FieldNameStyle:   styleAttrs("bold", "underline"),
				FieldValueStyle:  defaultStyle(),
				FooterStyle:      styleAttrs("dim", "italic"),
				URLStyle: Style{
					Style: tcell.StyleDefault.Foreground(tcell.GetColor("blue")).Underline(tcell.UnderlineStyleSolid),
				},
			},
		},
		MentionsList: MentionsListTheme{
			MinWidth:  20,
			MaxHeight: 0,
		},
		Dialog: DialogTheme{
			Style:           defaultStyle(),
			BackgroundStyle: styleAttrs("dim"),
		},
		Help: HelpTheme{
			ShortKeyStyle:  styleAttrs("dim"),
			ShortDescStyle: defaultStyle(),
			FullKeyStyle:   styleAttrs("dim"),
			FullDescStyle:  defaultStyle(),
		},
	}
}

func defaultStyle() Style {
	return Style{Style: tcell.StyleDefault}
}

func styleColor(color string) Style {
	return Style{Style: tcell.StyleDefault.Foreground(tcell.GetColor(color))}
}

func styleColorAttrs(color string, attrs ...string) Style {
	s := styleColor(color)
	for _, attr := range attrs {
		s.parseAttr(attr)
	}
	return s
}

func styleAttrs(attrs ...string) Style {
	s := defaultStyle()
	for _, attr := range attrs {
		s.parseAttr(attr)
	}
	return s
}
