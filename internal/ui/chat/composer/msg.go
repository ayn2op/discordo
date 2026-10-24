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
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
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

// edit is a text of the composer and the cursor in it.
type edit struct {
	text   string
	cursor int
}

// editMsg is the text and cursor that typing in the composer left.
type editMsg edit

func editLastMessage() tview.Cmd {
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

		files := openFiles(paths)
		if len(files) == 0 {
			return nil
		}

		return filesPickedMsg{channelID: channelID, files: files}
	}
}

// openFiles opens the files at paths, skipping those that fail to open.
func openFiles(paths []string) []sendpart.File {
	files := make([]sendpart.File, 0, len(paths))
	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			slog.Error("failed to open file", "path", path, "err", err)
			continue
		}

		files = append(files, sendpart.File{Name: filepath.Base(path), Reader: file})
	}

	return files
}

func closeFiles(files []sendpart.File) tview.Cmd {
	return func() tview.Msg {
		for _, file := range files {
			closeFile(file)
		}

		return nil
	}
}

func closeFile(file sendpart.File) {
	closer, ok := file.Reader.(io.Closer)
	if !ok {
		return
	}

	if err := closer.Close(); err != nil {
		slog.Error("failed to close file", "err", err)
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
		if err := m.state.Typing(channelID); err != nil {
			slog.Error("failed to send typing indicator", "err", err)
		}
		return nil
	}
}

func (m *Model) send() tview.Cmd {
	selectedChannel := m.channel
	if selectedChannel == nil {
		return nil
	}

	text := strings.TrimSpace(m.text)
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
			m.editMessage(*editing, text)
			return nil
		}

		data.Content = text
		m.sendMessage(selectedChannel.ID, data)
		return nil
	})
}

func (m Model) editMessage(message discord.Message, text string) {
	data := api.EditMessageData{Content: option.SomeNullable(text)}
	if _, err := m.state.EditMessageComplex(message.ChannelID, message.ID, data); err != nil {
		slog.Error("failed to edit message", "err", err)
	}
}

func (m Model) sendMessage(channelID discord.ChannelID, data api.SendMessageData) {
	if _, err := m.state.SendMessageComplex(channelID, data); err != nil {
		slog.Error("failed to send message in channel", "channel_id", channelID, "err", err)
	}
}

func (m *Model) tabComplete() tview.Cmd {
	end, name, previous := wordBeforeCursor(m.text, m.cursor, isMentionChar)
	if previous != '@' {
		m.stopTabCompletion()
		return nil
	}

	start := end - (len(name) + 1)
	channel := m.channel
	if channel == nil {
		return nil
	}

	if m.cfg.AutocompleteLimit == 0 {
		return m.completeBestMatch(channel, name, start, end)
	}

	if m.mentionsList.ItemCount() == 0 {
		return nil
	}

	name, ok := m.mentionsList.SelectedInsertText()
	if !ok {
		return nil
	}

	m.replace(start, end, "@"+name+" ")
	m.stopTabCompletion()
	return nil
}

// completeBestMatch replaces the name between start and end with the user that matches it best, as there is no mentions list to choose from.
func (m *Model) completeBestMatch(channel *discord.Channel, name string, start, end int) tview.Cmd {
	guildID := channel.GuildID
	if !guildID.IsValid() {
		users := channel.DMRecipients
		if matches := fuzzy.FindFrom(name, userList(users)); len(matches) > 0 {
			m.replace(start, end, "@"+users[matches[0].Index].Username+" ")
		}

		return nil
	}

	cmd := m.searchMember(guildID, name)
	members, err := m.state.Cabinet.Members(guildID)
	if err != nil {
		slog.Error("failed to get members from state", "guild_id", guildID, "err", err)
		return cmd
	}

	for _, match := range fuzzy.FindFrom(name, memberList(members)) {
		if channelHasUser(m.state, channel.ID, members[match.Index].User.ID) {
			m.replace(start, end, "@"+members[match.Index].User.Username+" ")
			return cmd
		}
	}

	return cmd
}

func (m *Model) tabSuggest() tview.Cmd {
	_, name, previous := wordBeforeCursor(m.text, m.cursor, isMentionChar)
	if previous != '@' {
		m.stopTabCompletion()
		return nil
	}

	channel := m.channel
	if channel == nil {
		return nil
	}

	m.mentionsList.Clear()

	var searchCmd tview.Cmd
	switch {
	case name == "":
		m.suggestRecentAuthors(channel)
	case !channel.GuildID.IsValid():
		m.suggestRecipients(channel, name)
	default:
		searchCmd = m.suggestMembers(channel, name)
	}

	if m.mentionsList.ItemCount() == 0 {
		m.stopTabCompletion()
		return searchCmd
	}

	m.mentionsList.Rebuild()
	return nil
}

// suggestRecipients suggests the recipients of the DM channel and the user that match name, as DMs have recipients, not members.
func (m *Model) suggestRecipients(channel *discord.Channel, name string) {
	me, _ := m.state.Cabinet.Me()
	users := append(slices.Clone(channel.DMRecipients), *me)
	for _, match := range fuzzy.FindFrom(name, userList(users)) {
		m.addMentionUser(&users[match.Index])
	}
}

// suggestMembers suggests the members who can see channel that match name, and returns the search for more of them.
func (m *Model) suggestMembers(channel *discord.Channel, name string) tview.Cmd {
	searchCmd := m.searchMember(channel.GuildID, name)
	members, err := m.state.Cabinet.Members(channel.GuildID)
	if err != nil {
		slog.Error("fetching members failed", "err", err)
		return searchCmd
	}

	matches := fuzzy.FindFrom(name, memberList(members))
	for _, match := range matches[:min(len(matches), int(m.cfg.AutocompleteLimit))] {
		if channelHasUser(m.state, channel.ID, members[match.Index].User.ID) && m.addMentionMember(channel.GuildID, &members[match.Index]) {
			break
		}
	}

	return searchCmd
}

// searchMember performs member discovery in a command goroutine.
// It emits a follow-up suggestion message once results are loaded.
func (m *Model) searchMember(guildID discord.GuildID, name string) tview.Cmd {
	if name == "" {
		return nil
	}

	key := guildID.String() + " " + name
	if _, ok := m.memberSearchCache[key]; ok {
		return nil
	}
	// If searching for "ab" returns less than SearchLimit, then "abc" would not return anything new because we already searched everything starting with "ab".
	// This will still be true even if a new member joins because arikawa loads new members into the state.
	if count, ok := m.memberSearchCache[key[:len(key)-1]]; ok && count < m.state.MemberState.SearchLimit {
		m.memberSearchCache[key] = count
		return nil
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
			GuildIDs:  []discord.GuildID{guildID},
			Query:     option.Some(name),
			Presences: m.state.MemberState.RequestPresences,
			Limit:     m.state.MemberState.SearchLimit,
			Nonce:     nonce,
		}); err != nil {
			slog.Error("failed to search guild members", "err", err, "guild_id", guildID, "query", name)
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
	text, cfg := m.text, m.cfg
	return tview.Suspend(func() tview.Msg {
		return runEditor(cfg, text)
	})
}

// runEditor opens text in the editor and returns what it was edited to.
func runEditor(cfg *config.Config, text string) tview.Msg {
	file, err := os.CreateTemp("", tmpFilePattern)
	if err != nil {
		slog.Error("failed to create tmp file", "err", err)
		return nil
	}

	name := file.Name()
	defer func() {
		if err := os.Remove(name); err != nil {
			slog.Error("failed to remove tmp file", "err", err)
		}
	}()

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

	edited, err := os.ReadFile(name)
	if err != nil {
		slog.Error("failed to read tmp file", "name", name, "err", err)
		return nil
	}

	return editorMsg(strings.TrimSpace(string(edited)))
}
