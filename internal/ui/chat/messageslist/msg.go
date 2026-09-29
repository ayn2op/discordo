package messageslist

import (
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/picker"
)

// Msg is implemented by the messages the list sends itself, which must reach it whatever has the focus.
type Msg interface{ messagesList() }

type olderMessagesLoadedMsg struct {
	ChannelID discord.ChannelID
	Older     []discord.Message
}

type deleteMessageMsg discord.Message

type attachmentActionMsg struct{ Action tview.Cmd }

func (olderMessagesLoadedMsg) messagesList() {}
func (deleteMessageMsg) messagesList()       {}
func (attachmentActionMsg) messagesList()    {}

// ReplyMsg asks to reply to Message by Name, mentioning its author if Mention is set.
type ReplyMsg struct {
	Message discord.Message
	Name    string
	Mention bool
}

// EditMsg asks to edit a message.
type EditMsg discord.Message

// ShowAttachmentsMsg asks to pick one of the attachments and links of the selected message.
type ShowAttachmentsMsg picker.Items
