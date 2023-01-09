package config

type (
	MessagesTextKeysConfig struct {
		CopyContent string `yaml:"copy_content"`

		Reply        string `yaml:"reply"`
		ReplyMention string `yaml:"reply_mention"`
		SelectReply  string `yaml:"select_reply"`

		SelectPrevious string `yaml:"select_previous"`
		SelectNext     string `yaml:"select_next"`
		SelectFirst    string `yaml:"select_first"`
		SelectLast     string `yaml:"select_last"`
	}

	MessageInputKeysConfig struct {
		Send  string `yaml:"send"`
		Paste string `yaml:"paste"`

		LaunchEditor string `yaml:"launch_editor"`
	}

	KeysConfig struct {
		Cancel string `yaml:"cancel"`

		MessagesText MessagesTextKeysConfig `yaml:"messages_text"`
		MessageInput MessageInputKeysConfig `yaml:"message_input"`
	}
)
