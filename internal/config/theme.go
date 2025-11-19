package config

import (
	"errors"
	"strings"

	"github.com/gdamore/tcell/v2"
)

var errInvalidType = errors.New("invalid type")

type StyleWrapper struct{ tcell.Style }

func NewStyleWrapper(style tcell.Style) StyleWrapper {
	return StyleWrapper{Style: style}
}

func (sw *StyleWrapper) UnmarshalTOML(value any) error {
	switch value := value.(type) {
	case string:
		for part := range strings.SplitSeq(value, ";") {
			if part == "" {
				continue
			}

			kv := strings.Split(part, ":")
			if len(kv) != 2 {
				continue
			}

			key, val := strings.TrimSpace(kv[0]), strings.TrimSpace(kv[1])
			switch key {
			case "foreground":
				sw.Style = sw.Foreground(tcell.GetColor(val))
			case "background":
				sw.Style = sw.Background(tcell.GetColor(val))
			case "attributes":
				var attrs tcell.AttrMask
				for s := range strings.SplitSeq(val, ",") {
					attrs |= stringToAttrMask(strings.TrimSpace(s))
				}
				sw.Style = sw.Attributes(attrs)
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

type (
	GuildsTreeTheme struct {
		AutoExpandFolders bool   `toml:"auto_expand_folders" default:"true"`
		Graphics          bool   `toml:"graphics" default:"true"`
		GraphicsColor     string `toml:"graphics_color" default:"default"`
	}

	MessagesListTheme struct {
		ReplyIndicator     string       `toml:"reply_indicator" default:">"`
		ForwardedIndicator string       `toml:"forwarded_indicator" default:"<"`
		MentionStyle       StyleWrapper `toml:"mention_style" default:"foreground:blue;attributes:bold"`
		EmojiStyle         StyleWrapper `toml:"emoji_style" default:"foreground:green"`
		URLStyle           StyleWrapper `toml:"url_style" default:"foreground:blue;attributes:underline"`
		AttachmentStyle    StyleWrapper `toml:"attachment_style" default:"foreground:yellow"`
	}

	MentionsListTheme struct {
		MinWidth  uint `toml:"min_width" default:"20" description:"Minimum width of the mentions list."`
		MaxHeight uint `toml:"max_height" default:"0" description:"Maximum height of the mentions list."`
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
