package config

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
)

var errInvalidType = errors.New("invalid type")

var (
	attributes = enum[tcell.AttrMask]{
		{"bold", tcell.AttrBold},
		{"dim", tcell.AttrDim},
		{"italic", tcell.AttrItalic},
		{"blink", tcell.AttrBlink},
		{"reverse", tcell.AttrReverse},
		{"strikethrough", tcell.AttrStrikeThrough},
	}
	underlines = enum[tcell.UnderlineStyle]{
		{"", tcell.UnderlineStyleNone},
		{"solid", tcell.UnderlineStyleSolid},
		{"double", tcell.UnderlineStyleDouble},
		{"curly", tcell.UnderlineStyleCurly},
		{"dotted", tcell.UnderlineStyleDotted},
		{"dashed", tcell.UnderlineStyleDashed},
	}
)

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
				underline, err := underlines.parse(s)
				if err != nil {
					return err
				}
				sw.Style = sw.Underline(underline)
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
	if s == "underline" {
		sw.Style = sw.Underline(true)
		return
	}
	// An unknown attribute parses to no attribute.
	attr, _ := attributes.parse(s)
	sw.Style = sw.Attributes(sw.GetAttributes() | attr)
}

// MarshalTOML writes the style as the inline table UnmarshalTOML reads.
func (sw StyleWrapper) MarshalTOML() ([]byte, error) {
	var fields []string
	add := func(key string, value any) {
		b, _ := json.Marshal(value)
		fields = append(fields, key+" = "+string(b))
	}
	if c := sw.GetForeground(); c != tcell.ColorDefault {
		add("foreground", colorName(c))
	}
	if c := sw.GetBackground(); c != tcell.ColorDefault {
		add("background", colorName(c))
	}
	var attrs []string
	for _, a := range attributes {
		if sw.GetAttributes()&a.value != 0 {
			attrs = append(attrs, a.name)
		}
	}
	switch len(attrs) {
	case 0:
	case 1:
		add("attributes", attrs[0])
	default:
		add("attributes", attrs)
	}
	if u := sw.GetUnderlineStyle(); u != tcell.UnderlineStyleNone {
		name, _ := underlines.text(u)
		add("underline", string(name))
	}
	if c := sw.GetUnderlineColor(); c != tcell.ColorDefault {
		add("underline_color", colorName(c))
	}
	if fields == nil {
		return []byte("{}"), nil
	}
	return []byte("{ " + strings.Join(fields, ", ") + " }"), nil
}

// colorName returns the first of the names of c, or its hex code if it has none.
func colorName(c tcell.Color) string {
	var names []string
	for name, color := range tcell.ColorNames {
		if color == c {
			names = append(names, name)
		}
	}
	if names == nil {
		return c.CSS()
	}
	return slices.Min(names)
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
		// The style used while the widget is focused.
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
		Style StyleWrapper `toml:"style"`
		// The style of everything behind the dialog.
		BackgroundStyle StyleWrapper `toml:"background_style"`
	}

	ScrollBarTheme struct {
		TrackStyle StyleWrapper `toml:"track_style"`
		ThumbStyle StyleWrapper `toml:"thumb_style"`
	}

	Theme struct {
		Title  ThemeStyle `toml:"title"`
		Footer ThemeStyle `toml:"footer"`
		Border ThemeStyle `toml:"border"`
		// The status styles are those of direct-message recipients.
		GuildsTree CommonTreeTheme `toml:"guilds_tree"`
		// The status styles are those of the dot before each member.
		MembersTree  CommonTreeTheme   `toml:"members_tree"`
		ScrollBar    ScrollBarTheme    `toml:"scroll_bar"`
		MessagesList MessagesListTheme `toml:"messages_list"`
		MentionsList MentionsListTheme `toml:"mentions_list"`
		Dialog       DialogTheme       `toml:"dialog"`
		Help         HelpTheme         `toml:"help"`
	}
)

type CommonTreeTheme struct {
	// The color of the lines that give the tree its shape.
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

func defaultTheme() Theme {
	var (
		plain  = tcell.StyleDefault
		dim    = StyleWrapper{plain.Dim(true)}
		bold   = plain.Bold(true)
		italic = plain.Italic(true)
		blue   = plain.Foreground(tcell.ColorBlue)
		focus  = ThemeStyle{dim, StyleWrapper{bold.Foreground(tcell.ColorGreen)}}
		tree   = CommonTreeTheme{
			GraphicsColor: "default",
			OnlineStyle:   StyleWrapper{plain.Foreground(tcell.ColorGreen)},
			IdleStyle:     StyleWrapper{plain.Foreground(tcell.ColorYellow)},
			DNDStyle:      StyleWrapper{plain.Foreground(tcell.ColorRed)},
			OfflineStyle:  StyleWrapper{plain.Foreground(tcell.ColorGray)},
		}
	)
	return Theme{
		Title:       focus,
		Footer:      focus,
		Border:      focus,
		GuildsTree:  tree,
		MembersTree: tree,
		ScrollBar:   ScrollBarTheme{TrackStyle: dim},
		MessagesList: MessagesListTheme{
			MentionStyle:         StyleWrapper{blue.Bold(true)},
			EmojiStyle:           StyleWrapper{plain.Foreground(tcell.ColorGreen)},
			URLStyle:             StyleWrapper{blue},
			AttachmentStyle:      StyleWrapper{plain.Foreground(tcell.ColorYellow)},
			ReactionStyle:        dim,
			OwnReactionStyle:     StyleWrapper{bold},
			SelectedMessageStyle: StyleWrapper{plain.Reverse(true)},
			Embeds: MessagesListEmbedsTheme{
				ProviderStyle:    StyleWrapper{italic.Dim(true)},
				AuthorStyle:      StyleWrapper{italic},
				TitleStyle:       StyleWrapper{blue.Bold(true)},
				DescriptionStyle: dim,
				FieldNameStyle:   StyleWrapper{bold.Underline(true)},
				FooterStyle:      StyleWrapper{italic.Dim(true)},
				URLStyle:         StyleWrapper{blue.Underline(true)},
			},
		},
		MentionsList: MentionsListTheme{SelectedStyle: StyleWrapper{plain.Reverse(true)}},
		Dialog:       DialogTheme{BackgroundStyle: dim},
		Help:         HelpTheme{ShortKeyStyle: dim, FullKeyStyle: dim},
	}
}
