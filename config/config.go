package config

var General = struct {
	UserAgent          string
	Mouse              bool
	Notifications      bool
	FetchMessagesLimit int
}{
	UserAgent:          "Mozilla/5.0 (X11; Linux x86_64; rv:95.0) Gecko/20100101 Firefox/95.0",
	Mouse:              true,
	Notifications:      true,
	FetchMessagesLimit: 50,
}

var Keybindings = struct {
	FocusChannelsTreeView  []string
	FocusMessagesView      []string
	FocusMessageInputField []string

	SelectPreviousMessage       []string
	SelectNextMessage           []string
	SelectFirstMessage          []string
	SelectLastMessage           []string
	SelectMessageReference      []string
	ReplySelectedMessage        []string
	MentionReplySelectedMessage []string
	CopySelectedMessage         []string
}{
	FocusChannelsTreeView:  []string{"Alt+Left"},
	FocusMessagesView:      []string{"Alt+Right"},
	FocusMessageInputField: []string{"Alt+Down"},

	SelectPreviousMessage:       []string{"Up"},
	SelectNextMessage:           []string{"Down"},
	SelectFirstMessage:          []string{"Home"},
	SelectLastMessage:           []string{"End"},
	ReplySelectedMessage:        []string{"Rune[r]"},
	MentionReplySelectedMessage: []string{"Rune[R]"},
	CopySelectedMessage:         []string{"Rune[c]"},
	SelectMessageReference:      []string{"Rune[m]"},
}
