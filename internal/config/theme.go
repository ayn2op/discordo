package config

import (
	"errors"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
)

var (
	errInvalidType = errors.New("invalid type")
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

	for key, val := range m {
		switch key {
		case "foreground":
			s, ok := val.(string)
			if !ok {
				continue
			}

			color := tcell.GetColor(s)
			sw.Style = sw.Foreground(color)
		case "background":
			s, ok := val.(string)
			if !ok {
				continue
			}

			color := tcell.GetColor(s)
			sw.Style = sw.Background(color)
		case "attributes":
			var attrs tcell.AttrMask
			switch val := val.(type) {
			case string:
				attrs |= stringToAttrMask(val)
			case []any:
				for _, attr := range val {
					s, ok := attr.(string)
					if !ok {
						continue
					}

					attrs |= stringToAttrMask(s)
				}

			}

			sw.Style = sw.Attributes(attrs)
		}
	}

	return nil
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

	BorderTheme struct {
		ThemeStyle
		Enabled bool             `toml:"enabled"`
		Padding [4]int           `toml:"padding"`
		Set     BorderSetWrapper `toml:"set"`
	}

	GuildsTreeTheme struct {
		AutoExpandFolders bool `toml:"auto_expand_folders"`

		Graphics      bool   `toml:"graphics"`
		GraphicsColor string `toml:"graphics_color"`

		PrivateChannelStyle StyleWrapper `toml:"private_channel_style"`
		GuildStyle          StyleWrapper `toml:"guild_style"`
		ChannelStyle        StyleWrapper `toml:"channel_style"`
	}

	MessagesListTheme struct {
		ShowUserColors bool `toml:"show_user_colors"`

		ReplyIndicator     string `toml:"reply_indicator"`
		ForwardedIndicator string `toml:"forwarded_indicator"`

		AuthorStyle     StyleWrapper `toml:"author_style"`
		MentionStyle    StyleWrapper `toml:"mention_style"`
		EmojiStyle      StyleWrapper `toml:"emoji_style"`
		URLStyle        StyleWrapper `toml:"url_style"`
		AttachmentStyle StyleWrapper `toml:"attachment_style"`
	}

	MentionsListTheme struct {
		PreferNicknames bool   `toml:"prefer_nicknames"`
		ShowUsernames   bool   `toml:"show_usernames"`
		ShowUserColors  bool   `toml:"show_user_colors"`

		MinWidth  uint `toml:"min_width"`
		MaxHeight uint `toml:"max_height"`
	}

	Theme struct {
		PreferNicknames    bool   `toml:"prefer_nicknames"`
		PreferDisplayNames bool   `toml:"prefer_display_names"`
		BackgroundColor string `toml:"background_color"`

		Title  TitleTheme  `toml:"title"`
		Border BorderTheme `toml:"border"`

		GuildsTree   GuildsTreeTheme   `toml:"guilds_tree"`
		MessagesList MessagesListTheme `toml:"messages_list"`
		MentionsList MentionsListTheme `toml:"mentions_list"`
	}
)

func stringToAttrMask(s string) tcell.AttrMask {
	switch s {
	case "bold":
		return tcell.AttrBold
	case "blink":
		return tcell.AttrBlink
	case "reverse":
		return tcell.AttrReverse
	case "underline":
		return tcell.AttrUnderline
	case "dim":
		return tcell.AttrDim
	case "italic":
		return tcell.AttrItalic
	case "strikethrough":
		return tcell.AttrStrikeThrough
	default:
		return tcell.AttrNone
	}
}
