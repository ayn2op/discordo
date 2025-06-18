package config

import (
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
)

type BorderSetWrapper struct{ tview.BorderSet }

func (bw *BorderSetWrapper) UnmarshalTOML(v any) error {
	switch v.(string) {
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
	switch v.(string) {
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
		return nil
	}

	for k, v := range m {
		s, ok := v.(string)
		if !ok {
			continue
		}

		switch k {
		case "foreground":
			color := tcell.GetColor(s)
			sw.Style = sw.Foreground(color)
		case "background":
			color := tcell.GetColor(s)
			sw.Style = sw.Background(color)
		}
	}

	return nil
}

type (
	BorderTheme struct {
		Enabled bool             `toml:"enabled"`
		Padding [4]int           `toml:"padding"`
		Set     BorderSetWrapper `toml:"set"`

		Style       StyleWrapper `toml:"style"`
		ActiveStyle StyleWrapper `toml:"active_style"`
	}

	TitleTheme struct {
		Style       StyleWrapper     `toml:"style"`
		ActiveStyle StyleWrapper     `toml:"active_style"`
		Alignment   AlignmentWrapper `toml:"alignment"`
	}

	Theme struct {
		BackgroundColor string `toml:"background_color"`
		PreferNicks     bool `toml:"prefer_nicks"`
		ShowUsernames   bool `toml:"show_usernames"`

		Title        TitleTheme        `toml:"title"`
		Border       BorderTheme       `toml:"border"`
		GuildsTree   GuildsTreeTheme   `toml:"guilds_tree"`
		MessagesText MessagesTextTheme `toml:"messages_text"`
		Autocomplete AutocompleteTheme `toml:"autocomplete"`
	}

	GuildsTreeTheme struct {
		AutoExpandFolders bool `toml:"auto_expand_folders"`

		Graphics      bool   `toml:"graphics"`
		GraphicsColor string `toml:"graphics_color"`

		PrivateChannelColor string `toml:"private_channel_color"`
		GuildColor          string `toml:"guild_color"`
		ChannelColor        string `toml:"channel_color"`
	}

	MessagesTextTheme struct {
		ShowUsernameColors bool `toml:"show_user_colors"`

		ReplyIndicator     string `toml:"reply_indicator"`
		ForwardedIndicator string `toml:"forwarded_indicator"`

		AuthorStyle     StyleWrapper `toml:"author_style"`
		MentionStyle    StyleWrapper `toml:"mention_style"`
		EmojiStyle      StyleWrapper `toml:"emoji_style"`
		URLStyle        StyleWrapper `toml:"url_style"`
		AttachmentStyle StyleWrapper `toml:"attachment_style"`
	}

	AutocompleteTheme struct {
		ShowNicknames      bool `toml:"show_user_nicks"`
		ShowUsernames      bool `toml:"show_usernames"`
		ShowUsernameColors bool `toml:"show_user_colors"`

		MinWidth  uint `toml:"min_width"`
		MaxHeight uint `toml:"max_height"`
	}
)
