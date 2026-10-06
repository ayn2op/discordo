package composer

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/arikawa/v3/utils/json/option"
	"github.com/ayn2op/arikawa/v3/utils/sendpart"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/textarea"
	"github.com/ncruces/zenity"
	"github.com/sahilm/fuzzy"
	"golang.design/x/clipboard"
)

// TabSuggestMsg suggests mentions for the word before the cursor.
type TabSuggestMsg struct{}

// EditLastMsg asks to edit the user's last message.
type EditLastMsg struct{}

// SentMsg reports that the composer sent a message.
type SentMsg struct{}

type editorMsg string

// editMsg edits the composer's text.
type editMsg textarea.Change

func (m Model) editLastMessage() tview.Cmd {
	return func() tview.Msg { return EditLastMsg{} }
}

type imagePastedMsg []byte

func pasteImage() tview.Cmd {
	return func() tview.Msg {
		data, err := clipboard.Read(context.Background(), clipboard.FmtImage)
		if err != nil {
			slog.Error("failed to read from clipboard", "err", err)
			return nil
		}
		return imagePastedMsg(data)
	}
}

type filesPickedMsg struct {
	channelID discord.ChannelID
	files     []sendpart.File
}

func (m Model) pickFiles() tview.Cmd {
	selectedChannel := m.channel
	if selectedChannel == nil {
		return nil
	}
	channelID := selectedChannel.ID

	return func() tview.Msg {
		paths, err := zenity.SelectFileMultiple()
		if err != nil {
			slog.Error("failed to open file dialog", "err", err)
			return nil
		}

		files := make([]sendpart.File, 0, len(paths))
		for _, path := range paths {
			file, err := os.Open(path)
			if err != nil {
				slog.Error("failed to open file", "path", path, "err", err)
				continue
			}
			files = append(files, sendpart.File{Name: filepath.Base(path), Reader: file})
		}
		if len(files) == 0 {
			return nil
		}
		return filesPickedMsg{channelID: channelID, files: files}
	}
}

func closeFiles(files []sendpart.File) tview.Cmd {
	return func() tview.Msg {
		for _, file := range files {
			if closer, ok := file.Reader.(io.Closer); ok {
				closer.Close()
			}
		}
		return nil
	}
}

func (m *Model) sendTyping() tview.Cmd {
	if !m.cfg.TypingIndicator.Send {
		return nil
	}

	now := time.Now()
	if now.Before(m.typingUntil) {
		return nil
	}
	m.typingUntil = now.Add(TypingDuration)

	selectedChannel := m.channel
	if selectedChannel == nil {
		return nil
	}
	channelID := selectedChannel.ID
	return func() tview.Msg {
		m.state.Typing(channelID)
		return nil
	}
}

func (m *Model) send() tview.Cmd {
	selectedChannel := m.channel
	if selectedChannel == nil {
		return nil
	}

	text := strings.TrimSpace(m.editState.Value())
	if text == "" && len(m.sendMessageData.Files) == 0 {
		return nil
	}

	text = m.processText(selectedChannel, []byte(text))
	data := *m.sendMessageData
	data.Files = slices.Clone(data.Files)

	editing := m.editing
	m.typingUntil = time.Time{}
	m.reset()

	sent := func() tview.Msg { return SentMsg{} }
	return tview.Batch(sent, func() tview.Msg {
		defer closeFiles(data.Files)()
		if editing != nil {
			editData := api.EditMessageData{Content: option.SomeNullable(text)}
			if _, err := m.state.EditMessageComplex(editing.ChannelID, editing.ID, editData); err != nil {
				slog.Error("failed to edit message", "err", err)
			}
			return nil
		}
		data.Content = text
		if _, err := m.state.SendMessageComplex(selectedChannel.ID, data); err != nil {
			slog.Error("failed to send message in channel", "channel_id", selectedChannel.ID, "err", err)
		}
		return nil
	})
}

func (m *Model) tabComplete() tview.Cmd {
	posEnd, name, r := wordBeforeCursor(m.editState.Value(), m.editState.Cursor(), isMentionChar)
	if r != '@' {
		m.stopTabCompletion()
		return nil
	}
	pos := posEnd - (len(name) + 1)

	selectedChannel := m.channel
	if selectedChannel == nil {
		return nil
	}
	gID := selectedChannel.GuildID

	if m.cfg.AutocompleteLimit == 0 {
		if !gID.IsValid() {
			users := selectedChannel.DMRecipients
			res := fuzzy.FindFrom(name, userList(users))
			if len(res) > 0 {
				m.editState.Replace(pos, posEnd, "@"+users[res[0].Index].Username+" ")
			}
		} else {
			cmd := m.searchMember(gID, name)
			members, err := m.state.Cabinet.Members(gID)
			if err != nil {
				slog.Error("failed to get members from state", "guild_id", gID, "err", err)
				return cmd
			}

			res := fuzzy.FindFrom(name, memberList(members))
			for _, r := range res {
				if channelHasUser(m.state, selectedChannel.ID, members[r.Index].User.ID) {
					m.editState.Replace(pos, posEnd, "@"+members[r.Index].User.Username+" ")
					return cmd
				}
			}
			return cmd
		}
		return nil
	}
	if m.mentionsList.ItemCount() == 0 {
		return nil
	}
	name, ok := m.mentionsList.SelectedInsertText()
	if !ok {
		return nil
	}
	m.editState.Replace(pos, posEnd, "@"+name+" ")
	m.stopTabCompletion()
	return nil
}

func (m *Model) tabSuggest() tview.Cmd {
	_, name, r := wordBeforeCursor(m.editState.Value(), m.editState.Cursor(), isMentionChar)
	if r != '@' {
		m.stopTabCompletion()
		return nil
	}
	channel := m.channel
	if channel == nil {
		return nil
	}
	m.mentionsList.Clear()

	gID := channel.GuildID
	switch {
	case name == "":
		m.suggestRecentAuthors(channel)
	case !gID.IsValid():
		// DMs have recipients, not members.
		me, _ := m.state.Cabinet.Me()
		users := append(slices.Clone(channel.DMRecipients), *me)
		for _, r := range fuzzy.FindFrom(name, userList(users)) {
			m.addMentionUser(&users[r.Index])
		}
	default:
		searchCmd := m.searchMember(gID, name)
		mems, err := m.state.Cabinet.Members(gID)
		if err != nil {
			slog.Error("fetching members failed", "err", err)
			return searchCmd
		}
		res := fuzzy.FindFrom(name, memberList(mems))
		if len(res) > int(m.cfg.AutocompleteLimit) {
			res = res[:int(m.cfg.AutocompleteLimit)]
		}
		for _, r := range res {
			if channelHasUser(m.state, channel.ID, mems[r.Index].User.ID) &&
				m.addMentionMember(gID, &mems[r.Index]) {
				break
			}
		}
		if m.mentionsList.ItemCount() == 0 {
			m.stopTabCompletion()
			return searchCmd
		}
	}

	if m.mentionsList.ItemCount() == 0 {
		m.stopTabCompletion()
		return nil
	}
	m.mentionsList.Rebuild()
	return nil
}

// searchMember performs member discovery in a command goroutine.
// It emits a follow-up suggestion message once results are loaded.
func (m *Model) searchMember(gID discord.GuildID, name string) tview.Cmd {
	if name == "" {
		return nil
	}

	key := gID.String() + " " + name
	if _, ok := m.memberSearchCache[key]; ok {
		return nil
	}
	// If searching for "ab" returns less than SearchLimit, then "abc" would not return anything new because we already searched everything starting with "ab".
	// This will still be true even if a new member joins because arikawa loads new members into the state.
	if count, ok := m.memberSearchCache[key[:len(key)-1]]; ok {
		if count < m.state.MemberState.SearchLimit {
			m.memberSearchCache[key] = count
			return nil
		}
	}

	now := time.Now()
	// Rate limit on our side because we can't distinguish between a successful search and SearchMember not doing anything because of its internal rate limit that we can't detect
	if m.lastSearch.Add(m.state.MemberState.SearchFrequency).After(now) {
		return nil
	}

	m.lastSearch = now
	nonce := memberSearchNonce + key
	return func() tview.Msg {
		if err := m.state.SendGateway(context.Background(), &gateway.RequestGuildMembersCommand{
			GuildIDs:  []discord.GuildID{gID},
			Query:     option.Some(name),
			Presences: m.state.MemberState.RequestPresences,
			Limit:     m.state.MemberState.SearchLimit,
			Nonce:     nonce,
		}); err != nil {
			slog.Error("failed to search guild members", "err", err, "guild_id", gID, "query", name)
		}
		return nil
	}
}

func (m Model) openEditor() tview.Cmd {
	if m.cfg.Editor == "" {
		return func() tview.Msg {
			slog.Warn("Attempt to open file with editor, but no editor is set")
			return nil
		}
	}
	text := m.editState.Value()
	cfg := m.cfg
	return tview.Suspend(func() tview.Msg {
		file, err := os.CreateTemp("", tmpFilePattern)
		if err != nil {
			slog.Error("failed to create tmp file", "err", err)
			return nil
		}
		name := file.Name()
		defer os.Remove(name)
		_, _ = file.WriteString(text)
		_ = file.Close()

		cmd, err := cfg.EditorCommand(name)
		if err != nil {
			slog.Error("failed to create editor command", "err", err)
			return nil
		}
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			slog.Error("failed to run command", "args", cmd.Args, "err", err)
			return nil
		}
		msg, err := os.ReadFile(name)
		if err != nil {
			slog.Error("failed to read tmp file", "name", name, "err", err)
			return nil
		}
		return editorMsg(strings.TrimSpace(string(msg)))
	})
}
