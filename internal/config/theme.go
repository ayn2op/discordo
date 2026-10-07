package config

import (
	"errors"
	"fmt"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

var errInvalidType = errors.New("invalid type")

type StyleWrapper struct{ tview.Style }

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
				case "":
					sw.Style = sw.Underline(tcell.UnderlineStyleNone)
				case "solid":
					sw.Style = sw.Underline(tcell.UnderlineStyleSolid)
				case "double":
					sw.Style = sw.Underline(tcell.UnderlineStyleDouble)
				case "curly":
					sw.Style = sw.Underline(tcell.UnderlineStyleCurly)
				case "dotted":
					sw.Style = sw.Underline(tcell.UnderlineStyleDotted)
				case "dashed":
					sw.Style = sw.Underline(tcell.UnderlineStyleDashed)
				default:
					return fmt.Errorf("unknown value: %q", s)
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

type (
	HelpTheme struct {
		ShortKeyStyle  StyleWrapper `toml:"short_key_style"`
		ShortDescStyle StyleWrapper `toml:"short_desc_style"`
		FullKeyStyle   StyleWrapper `toml:"full_key_style"`
		FullDescStyle  StyleWrapper `toml:"full_desc_style"`
	}

	ThemeStyle struct {
		NormalStyle StyleWrapper `toml:"normal_style"`
		ActiveStyle StyleWrapper `toml:"active_style"`
	}

	MessagesListTheme struct {
		AuthorStyle      StyleWrapper `toml:"author_style"`
		MentionStyle     StyleWrapper `toml:"mention_style"`
		EmojiStyle       StyleWrapper `toml:"emoji_style"`
		URLStyle         StyleWrapper `toml:"url_style"`
		AttachmentStyle  StyleWrapper `toml:"attachment_style"`
		ReactionStyle    StyleWrapper `toml:"reaction_style"`
		OwnReactionStyle StyleWrapper `toml:"own_reaction_style"`

		MessageStyle         StyleWrapper `toml:"message_style"`
		SelectedMessageStyle StyleWrapper `toml:"selected_message_style"`

		Embeds MessagesListEmbedsTheme `toml:"embeds"`
	}

	MessagesListEmbedsTheme struct {
		ProviderStyle    StyleWrapper `toml:"provider_style"`
		AuthorStyle      StyleWrapper `toml:"author_style"`
		TitleStyle       StyleWrapper `toml:"title_style"`
		DescriptionStyle StyleWrapper `toml:"description_style"`
		FieldNameStyle   StyleWrapper `toml:"field_name_style"`
		FieldValueStyle  StyleWrapper `toml:"field_value_style"`
		FooterStyle      StyleWrapper `toml:"footer_style"`
		URLStyle         StyleWrapper `toml:"url_style"`
	}

	MentionsListTheme struct {
		SelectedStyle StyleWrapper `toml:"selected_style"`
	}

	DialogTheme struct {
		Style           StyleWrapper `toml:"style"`
		BackgroundStyle StyleWrapper `toml:"background_style"`
	}

	ScrollBarTheme struct {
		TrackStyle StyleWrapper `toml:"track_style"`
		ThumbStyle StyleWrapper `toml:"thumb_style"`
	}

	Theme struct {
		Title        ThemeStyle        `toml:"title"`
		Footer       ThemeStyle        `toml:"footer"`
		Border       ThemeStyle        `toml:"border"`
		GuildsTree   CommonTreeTheme   `toml:"guilds_tree"`
		MembersTree  CommonTreeTheme   `toml:"members_tree"`
		ScrollBar    ScrollBarTheme    `toml:"scroll_bar"`
		MessagesList MessagesListTheme `toml:"messages_list"`
		MentionsList MentionsListTheme `toml:"mentions_list"`
		Dialog       DialogTheme       `toml:"dialog"`
		Help         HelpTheme         `toml:"help"`
	}
)

type CommonTreeTheme struct {
	GraphicsColor string `toml:"graphics_color"`

	OnlineStyle  StyleWrapper `toml:"online_style"`
	IdleStyle    StyleWrapper `toml:"idle_style"`
	DNDStyle     StyleWrapper `toml:"dnd_style"`
	OfflineStyle StyleWrapper `toml:"offline_style"`
}

// StatusStyle returns the style for a user with status.
func (ctt CommonTreeTheme) StatusStyle(status discord.Status) tview.Style {
	switch status {
	case discord.OnlineStatus:
		return ctt.OnlineStyle.Style
	case discord.IdleStatus:
		return ctt.IdleStyle.Style
	case discord.DoNotDisturbStatus:
		return ctt.DNDStyle.Style
	default:
		return ctt.OfflineStyle.Style
	}
}
