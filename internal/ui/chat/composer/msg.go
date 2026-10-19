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
	"github.com/ayn2op/discordo/internal/command"
	"github.com/ayn2op/discordo/internal/ui/chat/commandslist"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
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

type botCommandsMsg struct {
	commands []command.Item
}

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
				if err := closer.Close(); err != nil {
					slog.Error("failed to close file", "err", err)
				}
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

	// Handle /nick command
	if strings.HasPrefix(text, "/nick ") || text == "/nick" {
		if selectedChannel.GuildID.IsValid() && m.state != nil && m.state.Client != nil {
			newNick := strings.TrimSpace(strings.TrimPrefix(text, "/nick"))
			gID := selectedChannel.GuildID
			m.reset()
			return func() tview.Msg {
				if err := m.state.Client.ModifyCurrentMember(gID, newNick); err != nil {
					slog.Error("failed to change nickname", "guild_id", gID, "err", err)
				}
				return SentMsg{}
			}
		}
	}

	if expanded, ok := command.ExpandBuiltin(text); ok {
		text = expanded
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
	if m.commandsListVisible() {
		return m.completeCommand()
	}
	if m.mentionsListVisible() {
		return m.completeMention()
	}
	return nil
}

func (m *Model) completeCommand() tview.Cmd {
	insertText, ok := m.commandsList.SelectedInsertText()
	if !ok {
		return nil
	}
	firstSpace := strings.Index(m.text, " ")
	endPos := m.cursor
	if firstSpace != -1 {
		endPos = firstSpace
	}
	m.replace(0, endPos, insertText)
	m.stopTabCompletion()
	return nil
}

func (m *Model) completeMention() tview.Cmd {
	posEnd, name, r := wordBeforeCursor(m.text, m.cursor, isMentionChar)
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
				m.replace(pos, posEnd, "@"+users[res[0].Index].Username+" ")
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
					m.replace(pos, posEnd, "@"+members[r.Index].User.Username+" ")
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
	m.replace(pos, posEnd, "@"+name+" ")
	m.stopTabCompletion()
	return nil
}

func (m *Model) tabSuggest() tview.Cmd {
	if strings.HasPrefix(m.text, "/") {
		firstSpace := strings.Index(m.text, " ")
		if firstSpace == -1 || m.cursor <= firstSpace {
			m.mentionsList.Clear()
			query := m.text[1:m.cursor]
			return m.suggestCommands(query)
		}
	}
	m.commandsList.Clear()

	_, name, r := wordBeforeCursor(m.text, m.cursor, isMentionChar)
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

func (m *Model) suggestCommands(query string) tview.Cmd {
	channel := m.channel
	if channel == nil {
		return nil
	}
	m.commandsList.Clear()

	var userPerms discord.Permissions
	if channel.GuildID.IsValid() && m.state != nil {
		if me, err := m.state.Cabinet.Me(); err == nil {
			if perms, err := m.state.Permissions(channel.ID, me.ID); err == nil {
				userPerms = perms
			}
		}
	} else {
		userPerms = discord.PermissionAll
	}

	limit := int(m.cfg.AutocompleteLimit)
	if limit <= 0 {
		limit = 20
	}

	results := command.Search(query, channel.ID, userPerms, m.botCommands, limit)
	for _, item := range results {
		style := tcell.StyleDefault
		display := "/" + item.Name
		if item.Usage != "" {
			display += " " + item.Usage
		}
		if item.Description != "" {
			display += " - " + item.Description
		}
		if item.Source != "" {
			display += " (" + item.Source + ")"
		}

		m.commandsList.Append(commandslist.Item{
			Name:        item.Name,
			Usage:       item.Usage,
			Description: item.Description,
			Source:      item.Source,
			InsertText:  "/" + item.Name + " ",
			DisplayText: display,
			Style:       style,
		})
	}

	var fetchCmd tview.Cmd
	if channel.GuildID.IsValid() && len(m.botCommands) == 0 {
		fetchCmd = m.fetchBotCommands(channel.ID, query)
	}

	if m.commandsList.ItemCount() == 0 {
		m.stopTabCompletion()
		return fetchCmd
	}
	m.commandsList.Rebuild()
	return fetchCmd
}

func (m *Model) fetchBotCommands(channelID discord.ChannelID, query string) tview.Cmd {
	if m.state == nil || m.state.Client == nil {
		return nil
	}
	client := m.state.Client
	return func() tview.Msg {
		items, err := command.FetchBotCommands(context.Background(), client, channelID, query)
		if err != nil {
			slog.Debug("failed to fetch bot commands", "err", err)
			return nil
		}
		return botCommandsMsg{commands: items}
	}
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
	text := m.text
	cfg := m.cfg
	return tview.Suspend(func() tview.Msg {
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
		msg, err := os.ReadFile(name)
		if err != nil {
			slog.Error("failed to read tmp file", "name", name, "err", err)
			return nil
		}
		return editorMsg(strings.TrimSpace(string(msg)))
	})
}
