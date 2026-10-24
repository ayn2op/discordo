package config

import (
	"encoding/json"
	"errors"
	"slices"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
	"github.com/gdamore/tcell/v3/color"
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

type Style struct{ tview.Style }

func (s *Style) UnmarshalTOML(value any) error {
	table, ok := value.(map[string]any)
	if !ok {
		return errInvalidType
	}

	// Reset on new styles
	s.Style = tcell.StyleDefault

	for key, value := range table {
		if key == "attributes" {
			s.parseAttributes(value)
			continue
		}

		name, ok := value.(string)
		if !ok {
			continue
		}

		if err := s.set(key, name); err != nil {
			return err
		}
	}

	return nil
}

// set applies the option key with the value name.
func (s *Style) set(key, name string) error {
	switch key {
	case "foreground":
		s.Style = s.Foreground(tcell.GetColor(name))
	case "background":
		s.Style = s.Background(tcell.GetColor(name))
	case "underline":
		underline, err := underlines.parse(name)
		if err != nil {
			return err
		}

		s.Style = s.Underline(underline)
	case "underline_color":
		s.Style = s.Underline(tcell.GetColor(name))
	}

	return nil
}

// parseAttributes applies one attribute name or a list of them.
func (s *Style) parseAttributes(value any) {
	switch value := value.(type) {
	case string:
		s.parseAttribute(value)
	case []any:
		for _, name := range stringValues(value) {
			s.parseAttribute(name)
		}
	}
}

func (s *Style) parseAttribute(name string) {
	if name == "underline" {
		s.Style = s.Underline(true)
		return
	}

	// An unknown attribute parses to no attribute.
	attr, _ := attributes.parse(name)
	s.Style = s.Attributes(s.GetAttributes() | attr) //nolint:staticcheck
}

// MarshalTOML writes the style as the inline table UnmarshalTOML reads.
func (s Style) MarshalTOML() ([]byte, error) {
	var fields []string
	add := func(key string, value any) {
		encoded, _ := json.Marshal(value)
		fields = append(fields, key+" = "+string(encoded))
	}

	if foreground := s.GetForeground(); foreground != tcell.ColorDefault {
		add("foreground", colorName(foreground))
	}

	if background := s.GetBackground(); background != tcell.ColorDefault {
		add("background", colorName(background))
	}

	switch names := s.attributeNames(); len(names) {
	case 0:
	case 1:
		add("attributes", names[0])
	default:
		add("attributes", names)
	}

	if underline := s.GetUnderlineStyle(); underline != tcell.UnderlineStyleNone {
		name, _ := underlines.text(underline)
		add("underline", string(name))
	}

	if underlineColor := s.GetUnderlineColor(); underlineColor != tcell.ColorDefault {
		add("underline_color", colorName(underlineColor))
	}

	if fields == nil {
		return []byte("{}"), nil
	}

	return []byte("{ " + strings.Join(fields, ", ") + " }"), nil
}

func (s Style) attributeNames() []string {
	var names []string
	for _, attr := range attributes {
		if s.GetAttributes()&attr.value != 0 {
			names = append(names, attr.name)
		}
	}

	return names
}

// colorName returns the first of the names of target, or its hex code if it has none.
func colorName(target tcell.Color) string {
	var names []string
	for name, color := range tcell.ColorNames {
		if color == target {
			names = append(names, name)
		}
	}

	if names == nil {
		return target.CSS()
	}

	return slices.Min(names)
}

type (
	HelpThemeConfig struct {
		ShortKeyStyle  Style `toml:"short_key_style"`
		ShortDescStyle Style `toml:"short_desc_style"`
		FullKeyStyle   Style `toml:"full_key_style"`
		FullDescStyle  Style `toml:"full_desc_style"`
	}

	StyleThemeConfig struct {
		NormalStyle Style `toml:"normal_style"`
		// The style used while the widget is focused.
		ActiveStyle Style `toml:"active_style"`
	}

	MessagesListThemeConfig struct {
		AuthorStyle      Style `toml:"author_style"`
		MentionStyle     Style `toml:"mention_style"`
		EmojiStyle       Style `toml:"emoji_style"`
		URLStyle         Style `toml:"url_style"`
		AttachmentStyle  Style `toml:"attachment_style"`
		ReactionStyle    Style `toml:"reaction_style"`
		OwnReactionStyle Style `toml:"own_reaction_style"`

		MessageStyle         Style `toml:"message_style"`
		SelectedMessageStyle Style `toml:"selected_message_style"`

		Embeds MessagesListEmbedsThemeConfig `toml:"embeds"`
	}

	MessagesListEmbedsThemeConfig struct {
		ProviderStyle    Style `toml:"provider_style"`
		AuthorStyle      Style `toml:"author_style"`
		TitleStyle       Style `toml:"title_style"`
		DescriptionStyle Style `toml:"description_style"`
		FieldNameStyle   Style `toml:"field_name_style"`
		FieldValueStyle  Style `toml:"field_value_style"`
		FooterStyle      Style `toml:"footer_style"`
		URLStyle         Style `toml:"url_style"`
	}

	AutocompleteThemeConfig struct {
		SelectedStyle Style `toml:"selected_style"`
	}

	DialogThemeConfig struct {
		Style Style `toml:"style"`
		// The style of everything behind the dialog.
		BackgroundStyle Style `toml:"background_style"`
	}

	ScrollBarThemeConfig struct {
		TrackStyle Style `toml:"track_style"`
		ThumbStyle Style `toml:"thumb_style"`
	}

	ThemeConfig struct {
		Title  StyleThemeConfig `toml:"title"`
		Footer StyleThemeConfig `toml:"footer"`
		Border StyleThemeConfig `toml:"border"`
		// The status styles are those of direct-message recipients.
		GuildsTree GuildsTreeThemeConfig `toml:"guilds_tree"`
		// The status styles are those of the dot before each member.
		MembersTree  TreeThemeConfig         `toml:"members_tree"`
		ScrollBar    ScrollBarThemeConfig    `toml:"scroll_bar"`
		MessagesList MessagesListThemeConfig `toml:"messages_list"`
		Autocomplete AutocompleteThemeConfig `toml:"autocomplete"`
		Dialog       DialogThemeConfig       `toml:"dialog"`
		Help         HelpThemeConfig         `toml:"help"`
	}
)

type GuildsTreeThemeConfig struct {
	TreeThemeConfig
	// The style of whoever is speaking in the voice channel joined.
	SpeakingStyle Style `toml:"speaking_style"`
	// The style of how full a voice channel that has a user limit is.
	VoiceLimitStyle Style `toml:"voice_limit_style"`
}

type TreeThemeConfig struct {
	// The color of the lines that give the tree its shape.
	GraphicsColor string `toml:"graphics_color"`

	OnlineStyle  Style `toml:"online_style"`
	IdleStyle    Style `toml:"idle_style"`
	DNDStyle     Style `toml:"dnd_style"`
	OfflineStyle Style `toml:"offline_style"`
}

// StatusStyle returns the style for a user with status.
func (t TreeThemeConfig) StatusStyle(status discord.Status) tview.Style {
	switch status {
	case discord.OnlineStatus:
		return t.OnlineStyle.Style
	case discord.IdleStatus:
		return t.IdleStyle.Style
	case discord.DoNotDisturbStatus:
		return t.DNDStyle.Style
	default:
		return t.OfflineStyle.Style
	}
}

func defaultTheme() ThemeConfig {
	var (
		plain  = tcell.StyleDefault
		dim    = Style{plain.Dim(true)}
		bold   = plain.Bold(true)
		italic = plain.Italic(true)
		blue   = plain.Foreground(color.Blue)
		focus  = StyleThemeConfig{dim, Style{bold.Foreground(color.Green)}}
		tree   = TreeThemeConfig{
			GraphicsColor: "default",
			OnlineStyle:   Style{plain.Foreground(color.Green)},
			IdleStyle:     Style{plain.Foreground(color.Yellow)},
			DNDStyle:      Style{plain.Foreground(color.Red)},
			OfflineStyle:  Style{plain.Foreground(color.Gray)},
		}
	)
	return ThemeConfig{
		Title:  focus,
		Footer: focus,
		Border: focus,
		GuildsTree: GuildsTreeThemeConfig{
			TreeThemeConfig: tree,
			SpeakingStyle:   Style{plain.Foreground(color.Green)},
			VoiceLimitStyle: Style{plain.Foreground(color.DodgerBlue)},
		},
		MembersTree: tree,
		ScrollBar:   ScrollBarThemeConfig{TrackStyle: dim},
		MessagesList: MessagesListThemeConfig{
			MentionStyle:         Style{blue.Bold(true)},
			EmojiStyle:           Style{plain.Foreground(color.Green)},
			URLStyle:             Style{blue},
			AttachmentStyle:      Style{plain.Foreground(color.Yellow)},
			ReactionStyle:        dim,
			OwnReactionStyle:     Style{bold},
			SelectedMessageStyle: Style{plain.Reverse(true)},
			Embeds: MessagesListEmbedsThemeConfig{
				ProviderStyle:    Style{italic.Dim(true)},
				AuthorStyle:      Style{italic},
				TitleStyle:       Style{blue.Bold(true)},
				DescriptionStyle: dim,
				FieldNameStyle:   Style{bold.Underline(true)},
				FooterStyle:      Style{italic.Dim(true)},
				URLStyle:         Style{blue.Underline(true)},
			},
		},
		Autocomplete: AutocompleteThemeConfig{SelectedStyle: Style{plain.Reverse(true)}},
		Dialog:       DialogThemeConfig{BackgroundStyle: dim},
		Help:         HelpThemeConfig{ShortKeyStyle: dim, FullKeyStyle: dim},
	}
}
