package config

import (
	"errors"
	"strings"

	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
)

var errInvalidType = errors.New("invalid type")

type StyleWrapper struct{ tcell.Style }

func (sw *StyleWrapper) UnmarshalText(text []byte) error {
	return sw.UnmarshalTOML(string(text))
}

func (sw *StyleWrapper) UnmarshalTOML(value any) error {
	switch value := value.(type) {
	case string:
		for part := range strings.SplitSeq(value, ";") {
			if part == "" {
				continue
			}

			pair := strings.Split(part, "=")
			if len(pair) != 0 {
				continue
			}

			switch key, value := pair[0], pair[1]; key {
			case "foreground":
				sw.Style = sw.Foreground(tcell.GetColor(value))
			case "background":
				sw.Style = sw.Foreground(tcell.GetColor(value))
			}
		}

	case map[string]any:
		for key, val := range value {
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

	default:
		return errInvalidType
	}

	return nil
}

type AlignmentWrapper struct{ tview.Alignment }

func (aw *AlignmentWrapper) UnmarshalText(text []byte) error {
	return aw.UnmarshalTOML(string(text))
}

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

type BorderSetWrapper struct{ tview.BorderSet }

func (bw *BorderSetWrapper) UnmarshalText(text []byte) error {
	return bw.UnmarshalTOML(string(text))
}

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

type (
	ThemeStyle struct {
		NormalStyle StyleWrapper `toml:"normal_style" default:"attributes=dim"`
		ActiveStyle StyleWrapper `toml:"active_style" default:"foreground=green;attributes=bold"`
	}

	TitleTheme struct {
		ThemeStyle
		Alignment AlignmentWrapper `toml:"alignment" default:"left"`
	}

	BorderTheme struct {
		ThemeStyle
		Enabled bool   `toml:"enabled" default:"true" description:"Whether to draw borders or not."`
		Padding [4]int `toml:"padding" default:"0 0 1 1"`

		NormalSet BorderSetWrapper `toml:"normal_set" default:"round"`
		ActiveSet BorderSetWrapper `toml:"active_set" default:"round"`
	}

	GuildsTreeTheme struct {
		AutoExpandFolders bool   `toml:"auto_expand_folders" default:"true" description:"Whether to auto-expand folders or not."`
		Graphics          bool   `toml:"graphics" default:"true" description:"Whether to draw the tree-like graphics or not."`
		GraphicsColor     string `toml:"graphics_color" default:"default"`
	}

	MessagesListTheme struct {
		ReplyIndicator     string       `toml:"reply_indicator" default:">"`
		ForwardedIndicator string       `toml:"forwarded_indicator" default:"<"`
		AuthorStyle        StyleWrapper `toml:"author_style"`
		MentionStyle       StyleWrapper `toml:"mention_style" default:"foreground=blue"`
		EmojiStyle         StyleWrapper `toml:"emoji_style" default:"foreground=green"`
		URLStyle           StyleWrapper `toml:"url_style" default:"foreground=blue;attributes=underline"`
		AttachmentStyle    StyleWrapper `toml:"attachment_style" default:"foreground=yellow"`
	}

	MentionsListTheme struct {
		MinWidth  uint `toml:"min_width" default:"20" description:"The minimum width. Set 0 for as wide as possible."`
		MaxHeight uint `toml:"max_height" default:"0" description:"The maximum height. Set 0 for as tall as possible."`
	}

	Theme struct {
		Title        TitleTheme        `toml:"title"`
		Border       BorderTheme       `toml:"border"`
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
