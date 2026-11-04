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
	"github.com/ayn2op/discordo/internal/ui/autocomplete/emojis"
	"github.com/ayn2op/discordo/internal/ui/autocomplete/mentions"
	"github.com/ayn2op/tview"
	"github.com/ncruces/zenity"
	"golang.design/x/clipboard"
)

// RefreshAutocompleteMsg refreshes the autocomplete.
type RefreshAutocompleteMsg struct{}

// EditLastMsg asks to edit the user's last message.
type EditLastMsg struct{}

// SentMsg reports that the composer sent a message.
type SentMsg struct{}

// DeleteEditedMsg asks to delete the message being edited, which was cleared rather than edited.
type DeleteEditedMsg discord.Message

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
		// Submitting a message that was cleared rather than edited deletes it, as in the Discord client.
		if m.editing == nil {
			return nil
		}

		message := *m.editing
		return func() tview.Msg { return DeleteEditedMsg(message) }
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

func (m *Model) complete() tview.Cmd {
	if item, ok := m.autocomplete.Selected(); ok {
		_, name := m.query()
		m.replace(m.cursor-len(name)-1, m.cursor, item.Insert()+" ")
	}

	return m.showCompletions("", nil)
}

func (m *Model) showAutocomplete() tview.Cmd {
	trigger, name := m.query()
	if trigger == 0 || m.channel == nil {
		return m.showCompletions("", nil)
	}

	guildID, limit := m.channel.GuildID, int(m.cfg.UI.Autocomplete.Limit)
	if trigger == '@' {
		items, loadImages := mentions.Items(m.cfg, m.state, m.channel, name, limit)
		return tview.Batch(m.showCompletions(mentions.Title, items), loadImages, m.searchMember(guildID, name))
	}

	items, loadImages := emojis.Items(m.cfg, m.state, guildID, name, limit)
	return tview.Batch(m.showCompletions(emojis.Title, items), loadImages)
}

// searchMember asks Discord for members matching name; they arrive later as a gateway event.
func (m *Model) searchMember(guildID discord.GuildID, name string) tview.Cmd {
	if name == "" || !guildID.IsValid() {
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
