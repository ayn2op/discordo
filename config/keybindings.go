package config

type KeybindingsConfig struct {
	ToggleGuildsList        string `toml:"toggle_guilds_list"`
	ToggleChannelsTreeView  string `toml:"toggle_channels_tree_view"`
	ToggleMessagesTextView  string `toml:"toggle_messages_text_view"`
	ToggleMessageInputField string `toml:"toggle_message_input_field"`

	OpenMessageActionsList string `toml:"open_message_actions_list"`

	OpenExternalEditor     string `toml:"open_external_editor"`

	OpenAttachment         string `toml:"open_attachment"`
	DownloadAttachment     string `toml:"download_attachment"`

	SelectPreviousMessage string `toml:"select_previous_message"`
	SelectNextMessage     string `toml:"select_next_message"`
	SelectFirstMessage    string `toml:"select_first_message"`
	SelectLastMessage     string `toml:"select_last_message"`

}

func newKeybindingsConfig() KeybindingsConfig {
	return KeybindingsConfig{
		ToggleGuildsList:        "Rune[g]",
		ToggleChannelsTreeView:  "Rune[c]",
		ToggleMessagesTextView:  "Rune[m]",
		ToggleMessageInputField: "Rune[i]",

		OpenMessageActionsList: "Rune[a]",
		OpenExternalEditor:     "Ctrl+E",

		OpenAttachment: 		"Rune[o]",
		DownloadAttachment:		"Rune[d]",

		SelectPreviousMessage: "Up",
		SelectNextMessage:     "Down",
		SelectFirstMessage:    "Home",
		SelectLastMessage:     "End",
	}
}
