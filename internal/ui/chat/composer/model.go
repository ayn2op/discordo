package composer

import (
	"bytes"
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ayn2op/arikawa/v3/api"
	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/ayn2op/arikawa/v3/state"
	"github.com/ayn2op/arikawa/v3/utils/json/option"
	"github.com/ayn2op/arikawa/v3/utils/sendpart"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/chat/mentionslist"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/textarea"
	"github.com/gdamore/tcell/v3"
	"github.com/ncruces/zenity"
	"github.com/rivo/uniseg"
	"github.com/sahilm/fuzzy"
	"github.com/yuin/goldmark/ast"
	"golang.design/x/clipboard"
)

const (
	tmpFilePattern      = consts.Name + "_*.md"
	imageAttachmentName = "clipboard.png"
	memberSearchNonce   = "autocomplete:"
)

var mentionRegex = regexp.MustCompile("@[a-zA-Z0-9._]+")

type Model struct {
	title, footer string
	editState     textarea.EditState
	placeholder   string
	disabled      bool

	state *ningen.State
	// channel is the selected channel, or nil for none.
	channel *discord.Channel

	cfg *config.Config

	editing           *discord.Message
	sendMessageData   *api.SendMessageData
	memberSearchCache map[string]uint
	mentionsList      mentionslist.Model
	lastSearch        time.Time

	// height is the number of rows the composer takes, including its border.
	height int
	// mentionsVisible reports whether the mentions list is shown above the composer.
	mentionsVisible bool

	typingUntil time.Time
}

// TypingDuration is how long Discord shows someone as typing after they start.
const TypingDuration = 10 * time.Second

// TabSuggestMsg suggests mentions for the word before the cursor.
type TabSuggestMsg struct{}

// EditLastMsg asks to edit the user's last message.
type EditLastMsg struct{}

// SentMsg reports that the composer sent a message.
type SentMsg struct{}
type editorMsg string

// editMsg edits the composer's text.
type editMsg textarea.Change

var _ help.KeyMap = Model{}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	c := Model{
		placeholder:       "Select a channel to start chatting",
		disabled:          true,
		cfg:               cfg,
		state:             state,
		sendMessageData:   &api.SendMessageData{},
		memberSearchCache: make(map[string]uint),
		mentionsList:      mentionslist.NewModel(cfg),
		height:            3,
	}
	return c
}

// SetChannel sets the channel messages are sent to, disabling the composer if the user may not send messages there.
func (c *Model) SetChannel(channel *discord.Channel) {
	c.channel, c.typingUntil = channel, time.Time{}
	isDM := channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM
	c.disabled = !isDM && !c.state.HasPermissions(channel.ID, discord.PermissionSendMessages)
	c.placeholder = "Message..."
	if c.disabled {
		c.placeholder = "You do not have permission to send messages in this channel."
	}
}

// Height returns the number of rows the composer takes, including its border.
func (c *Model) Height() int {
	return c.height
}

func (c *Model) Disabled() bool {
	return c.disabled
}

// View shows the text area in a box, taking keys if focused.
func (c Model) View(focused bool) tview.Element { return view{&c, focused} }

// view is the composer as shown, with or without the focus.
type view struct {
	*Model
	focused bool
}

func (v view) Draw(screen tview.Screen, area tview.Rectangle) {
	v.box(v.focused).Draw(screen, area)
}

// Handle leaves bound keys to Update and passes other messages to the text area.
func (v view) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg {
	if key, ok := msg.(tview.KeyMsg); ok && v.isBound(key) {
		return msg
	}
	return v.box(v.focused).Handle(msg, area)
}

// box returns the text area in a box with the title and footer.
func (c *Model) box(focused bool) box.Widget {
	text := textarea.New(&c.editState).
		Placeholder(c.placeholder).
		Keybind(c.editAction).
		Focused(focused && !c.disabled).
		OnChange(func(a textarea.Change) tview.Msg { return editMsg(a) })
	return ui.Box(text, &c.cfg.Theme, focused).Title(c.title).Footer(c.footer)
}

// isBound reports whether key is one of the composer's keybinds.
func (c *Model) isBound(key tview.KeyMsg) bool {
	k := c.cfg.Keybinds.Composer
	if keybind.Matches(key, k.Paste.Keybind, k.Send.Keybind, k.OpenEditor.Keybind, k.OpenFilePicker.Keybind, k.Cancel.Keybind, k.TabComplete.Keybind, k.ToggleReplyMention.Keybind, k.Undo.Keybind) {
		return true
	}
	return keybind.Matches(key, k.EditLast.Keybind) && c.canEditLastMessage()
}

// insert inserts text at the cursor.
func (c *Model) insert(text string) {
	c.replace(c.editState.Cursor(), c.editState.Cursor(), text)
}

// pasteText inserts the text on the clipboard at the cursor.
func (c *Model) pasteText() {
	data, err := clipboard.Read(context.Background(), clipboard.FmtText)
	if err != nil {
		slog.Error("failed to read from clipboard", "err", err)
		return
	}
	c.insert(string(data))
}

// frame returns the cells the composer's border and padding take on each side.
func (c *Model) frame() (left, vertical int) {
	const size = 100
	inner := c.box(false).InnerArea(tview.Rectangle{Width: size, Height: size})
	return inner.X, size - inner.Height
}

func (c *Model) resizeForContent() {
	_, frame := c.frame()
	c.height = min(strings.Count(c.editState.Value(), "\n")+1, max(c.cfg.Composer.MaxHeight, 1)) + frame
}

func (c *Model) reset() {
	c.editing = nil
	c.sendMessageData = &api.SendMessageData{}
	c.setTitle("")
	c.setFooter("")
	c.setText("")
}

// setText replaces the text and moves the cursor to its end.
func (c *Model) setText(text string) {
	c.editState.SetValue(text)
	c.resizeForContent()
}

// replace replaces the bytes from start to end of the text and moves the cursor after it.
func (c *Model) replace(start, end int, text string) {
	c.editState.Replace(start, end, text)
	c.resizeForContent()
}

func (c *Model) setTitle(title string) {
	c.title = title
	c.resizeForContent()
}

func (c *Model) setFooter(footer string) {
	c.footer = footer
	c.resizeForContent()
}

func (c Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	cmd := c.update(msg)
	return c, cmd
}

// update changes c in response to msg and returns a command to run, or nil.
func (c *Model) update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case TabSuggestMsg:
		return c.tabSuggest()
	case imagePastedMsg:
		if len(msg) == 0 {
			return nil
		}
		c.attach(imageAttachmentName, bytes.NewReader(msg))
		return nil
	case filesPickedMsg:
		selectedChannel := c.channel
		if selectedChannel == nil || selectedChannel.ID != msg.channelID {
			return closeFiles(msg.files)
		}
		for _, file := range msg.files {
			c.attach(file.Name, file.Reader)
		}
		return nil
	case editorMsg:
		c.setText(string(msg))
		return nil
	case editMsg:
		c.editState.Apply(textarea.Change(msg))
		c.resizeForContent()
		typingCmd := c.sendTyping()
		if c.cfg.AutocompleteLimit > 0 {
			return tview.Batch(typingCmd, c.tabSuggest())
		}
		return typingCmd

	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.EditLast.Keybind) && c.canEditLastMessage():
			return c.editLastMessage()
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Paste.Keybind):
			c.pasteText()
			return pasteImage()
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Send.Keybind):
			if c.mentionsVisible {
				return c.tabComplete()
			}
			return c.send()
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.OpenEditor.Keybind):
			return tview.Sequence(c.stopTabCompletion(), c.openEditor())
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.OpenFilePicker.Keybind):
			return tview.Sequence(c.stopTabCompletion(), c.pickFiles())
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Cancel.Keybind):
			if c.mentionsVisible {
				return c.stopTabCompletion()
			}
			c.reset()
			return nil
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.TabComplete.Keybind):
			if c.mentionsVisible {
				return c.tabComplete()
			}
			c.insert("\t")
			return nil
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.ToggleReplyMention.Keybind):
			c.toggleReplyMention()
			return nil
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Undo.Keybind):
			c.editState.Undo()
			c.resizeForContent()
			return nil
		}
	case mentionslist.Msg:
		var cmd tview.Cmd
		c.mentionsList, cmd = c.mentionsList.Update(msg)
		return cmd
	}
	return nil
}

func (c *Model) canEditLastMessage() bool {
	return !c.Disabled() && c.editing == nil && c.editState.Value() == "" &&
		c.sendMessageData.Reference == nil && len(c.sendMessageData.Files) == 0 &&
		!c.mentionsVisible
}

func (c *Model) editLastMessage() tview.Cmd {
	return func() tview.Msg { return EditLastMsg{} }
}

// StartEdit puts message in the composer to be edited.
func (c *Model) StartEdit(message discord.Message) {
	c.editing = &message
	c.setTitle("Editing")
	c.setText(message.Content)
}

// StartReply makes the next message a reply to message by name, mentioning its author if mention is set.
func (c *Model) StartReply(message discord.Message, name string, mention bool) {
	data := c.sendMessageData
	data.Reference = &discord.MessageReference{MessageID: message.ID}
	data.AllowedMentions = &api.AllowedMentions{RepliedUser: option.Some(mention)}
	title := "Replying to "
	if mention {
		title = "[@] " + title
	}
	c.setTitle(title + name)
}

func (c *Model) toggleReplyMention() {
	data := c.sendMessageData
	if data.Reference == nil || data.AllowedMentions == nil || data.AllowedMentions.RepliedUser == nil {
		return
	}

	mention := !*data.AllowedMentions.RepliedUser
	data.AllowedMentions.RepliedUser = option.Some(mention)
	title := strings.TrimPrefix(c.title, "[@] ")
	if mention {
		title = "[@] " + title
	}
	c.setTitle(title)
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

func (c *Model) pickFiles() tview.Cmd {
	selectedChannel := c.channel
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

func (c *Model) sendTyping() tview.Cmd {
	if !c.cfg.TypingIndicator.Send {
		return nil
	}

	now := time.Now()
	if now.Before(c.typingUntil) {
		return nil
	}
	c.typingUntil = now.Add(TypingDuration)

	selectedChannel := c.channel
	if selectedChannel == nil {
		return nil
	}
	channelID := selectedChannel.ID
	return func() tview.Msg {
		c.state.Typing(channelID)
		return nil
	}
}

func (c *Model) send() tview.Cmd {
	selectedChannel := c.channel
	if selectedChannel == nil {
		return nil
	}

	text := strings.TrimSpace(c.editState.Value())
	if text == "" && len(c.sendMessageData.Files) == 0 {
		return nil
	}

	text = c.processText(selectedChannel, []byte(text))
	data := *c.sendMessageData
	data.Files = slices.Clone(data.Files)

	editing := c.editing
	c.typingUntil = time.Time{}
	c.reset()

	sent := func() tview.Msg { return SentMsg{} }
	return tview.Batch(sent, func() tview.Msg {
		defer closeFiles(data.Files)()
		if editing != nil {
			editData := api.EditMessageData{Content: option.SomeNullable(text)}
			if _, err := c.state.EditMessageComplex(editing.ChannelID, editing.ID, editData); err != nil {
				slog.Error("failed to edit message", "err", err)
			}
			return nil
		}
		data.Content = text
		if _, err := c.state.SendMessageComplex(selectedChannel.ID, data); err != nil {
			slog.Error("failed to send message in channel", "channel_id", selectedChannel.ID, "err", err)
		}
		return nil
	})
}

func (c *Model) processText(channel *discord.Channel, src []byte) string {
	// Fast path: no mentions to expand.
	if bytes.IndexByte(src, '@') == -1 {
		return string(src)
	}

	// Fast path: no back ticks (code blocks), so expand mentions directly.
	if bytes.IndexByte(src, '`') == -1 {
		return string(c.expandMentions(channel, src))
	}

	var (
		ranges     [][2]int
		canMention = true
	)

	ast.Walk(md.Parse(src), func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		switch node := node.(type) {
		case *ast.CodeBlock, *ast.FencedCodeBlock:
			canMention = !enter
		case *md.Inline:
			if (node.Attr & md.AttrMonospace) != 0 {
				canMention = !enter
			}
		case *ast.Text:
			if canMention {
				ranges = append(ranges, [2]int{node.Segment.Start,
					node.Segment.Stop})
			}
		}
		return ast.WalkContinue, nil
	})

	for _, rng := range ranges {
		src = slices.Replace(src, rng[0], rng[1], c.expandMentions(channel, src[rng[0]:rng[1]])...)
	}

	return string(src)
}

func (c *Model) expandMentions(channel *discord.Channel, src []byte) []byte {
	state := c.state
	return mentionRegex.ReplaceAllFunc(src, func(input []byte) []byte {
		output := input
		name := string(input[1:])
		if channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM {
			for _, user := range channel.DMRecipients {
				if strings.EqualFold(user.Username, name) {
					return []byte(user.ID.Mention())
				}
			}
			// self ping
			me, _ := state.Cabinet.Me()
			if strings.EqualFold(me.Username, name) {
				return []byte(me.ID.Mention())
			}
			return output
		}
		state.MemberStore.Each(channel.GuildID, func(m *discord.Member) bool {
			if strings.EqualFold(m.User.Username, name) {
				if channelHasUser(state, channel.ID, m.User.ID) {
					output = []byte(m.User.ID.Mention())
				}
				return true
			}
			return false
		})
		return output
	})
}

// wordBeforeCursor returns cursor, the runes satisfying f just before it, and the rune before those.
func wordBeforeCursor(value string, cursor int, f func(rune) bool) (int, string, rune) {
	for start := cursor; start > 0; {
		r, size := utf8.DecodeLastRuneInString(value[:start])
		if !f(r) {
			return cursor, value[start:cursor], r
		}
		start -= size
	}
	return cursor, value[:cursor], 0
}

func isMentionChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
}

func (c *Model) tabComplete() tview.Cmd {
	posEnd, name, r := wordBeforeCursor(c.editState.Value(), c.editState.Cursor(), isMentionChar)
	if r != '@' {
		return c.stopTabCompletion()
	}
	pos := posEnd - (len(name) + 1)

	selectedChannel := c.channel
	if selectedChannel == nil {
		return nil
	}
	gID := selectedChannel.GuildID

	if c.cfg.AutocompleteLimit == 0 {
		if !gID.IsValid() {
			users := selectedChannel.DMRecipients
			res := fuzzy.FindFrom(name, userList(users))
			if len(res) > 0 {
				c.replace(pos, posEnd, "@"+users[res[0].Index].Username+" ")
			}
		} else {
			cmd := c.searchMember(gID, name)
			members, err := c.state.Cabinet.Members(gID)
			if err != nil {
				slog.Error("failed to get members from state", "guild_id", gID, "err", err)
				return cmd
			}

			res := fuzzy.FindFrom(name, memberList(members))
			for _, r := range res {
				if channelHasUser(c.state, selectedChannel.ID, members[r.Index].User.ID) {
					c.replace(pos, posEnd, "@"+members[r.Index].User.Username+" ")
					return cmd
				}
			}
			return cmd
		}
		return nil
	}
	if c.mentionsList.ItemCount() == 0 {
		return nil
	}
	name, ok := c.mentionsList.SelectedInsertText()
	if !ok {
		return nil
	}
	c.replace(pos, posEnd, "@"+name+" ")
	return c.stopTabCompletion()
}

func (c *Model) tabSuggest() tview.Cmd {
	_, name, r := wordBeforeCursor(c.editState.Value(), c.editState.Cursor(), isMentionChar)
	if r != '@' {
		return c.stopTabCompletion()
	}
	channel := c.channel
	if channel == nil {
		return nil
	}
	c.mentionsList.Clear()

	gID := channel.GuildID
	switch {
	case name == "":
		c.suggestRecentAuthors(channel)
	case !gID.IsValid():
		// DMs have recipients, not members.
		me, _ := c.state.Cabinet.Me()
		users := append(slices.Clone(channel.DMRecipients), *me)
		for _, r := range fuzzy.FindFrom(name, userList(users)) {
			c.addMentionUser(&users[r.Index])
		}
	default:
		searchCmd := c.searchMember(gID, name)
		mems, err := c.state.Cabinet.Members(gID)
		if err != nil {
			slog.Error("fetching members failed", "err", err)
			return searchCmd
		}
		res := fuzzy.FindFrom(name, memberList(mems))
		if len(res) > int(c.cfg.AutocompleteLimit) {
			res = res[:int(c.cfg.AutocompleteLimit)]
		}
		for _, r := range res {
			if channelHasUser(c.state, channel.ID, mems[r.Index].User.ID) &&
				c.addMentionMember(gID, &mems[r.Index]) {
				break
			}
		}
		if c.mentionsList.ItemCount() == 0 {
			return tview.Batch(c.stopTabCompletion(), searchCmd)
		}
	}

	if c.mentionsList.ItemCount() == 0 {
		return c.stopTabCompletion()
	}
	c.mentionsList.Rebuild()
	return c.showMentionsList()
}

// suggestRecentAuthors suggests the authors of the channel's recent messages, other than the user.
func (c *Model) suggestRecentAuthors(channel *discord.Channel) {
	messages, err := c.state.Cabinet.Messages(channel.ID)
	if err != nil {
		return
	}
	me, _ := c.state.Cabinet.Me()
	shown := map[discord.UserID]bool{me.ID: true}
	for _, m := range messages {
		if shown[m.Author.ID] {
			continue
		}
		shown[m.Author.ID] = true
		if !channel.GuildID.IsValid() {
			c.addMentionUser(&m.Author)
			continue
		}
		c.state.MemberState.RequestMember(channel.GuildID, m.Author.ID)
		if member, err := c.state.Cabinet.Member(channel.GuildID, m.Author.ID); err == nil && c.addMentionMember(channel.GuildID, member) {
			return
		}
	}
}

type memberList []discord.Member
type userList []discord.User

func (ml memberList) String(i int) string {
	return ml[i].Nick + ml[i].User.DisplayName + ml[i].User.Tag()
}

func (ml memberList) Len() int {
	return len(ml)
}

func (ul userList) String(i int) string {
	return ul[i].DisplayName + ul[i].Tag()
}

func (ul userList) Len() int {
	return len(ul)
}

// channelHasUser checks if a user has permission to view the specified channel.
func channelHasUser(state *ningen.State, channelID discord.ChannelID, userID discord.UserID) bool {
	perms, err := state.Permissions(channelID, userID)
	if err != nil {
		slog.Error("failed to get permissions", "err", err, "channel", channelID, "user", userID)
		return false
	}
	return perms.Has(discord.PermissionViewChannel)
}

// searchMember performs member discovery in a command goroutine.
// It emits a follow-up suggestion message once results are loaded.
func (c *Model) searchMember(gID discord.GuildID, name string) tview.Cmd {
	if name == "" {
		return nil
	}

	key := gID.String() + " " + name
	if _, ok := c.memberSearchCache[key]; ok {
		return nil
	}
	// If searching for "ab" returns less than SearchLimit, then "abc" would not return anything new because we already searched everything starting with "ab".
	// This will still be true even if a new member joins because arikawa loads new members into the state.
	if count, ok := c.memberSearchCache[key[:len(key)-1]]; ok {
		if count < c.state.MemberState.SearchLimit {
			c.memberSearchCache[key] = count
			return nil
		}
	}

	now := time.Now()
	// Rate limit on our side because we can't distinguish between a successful search and SearchMember not doing anything because of its internal rate limit that we can't detect
	if c.lastSearch.Add(c.state.MemberState.SearchFrequency).After(now) {
		return nil
	}

	c.lastSearch = now
	nonce := memberSearchNonce + key
	return func() tview.Msg {
		if err := c.state.SendGateway(context.Background(), &gateway.RequestGuildMembersCommand{
			GuildIDs:  []discord.GuildID{gID},
			Query:     option.Some(name),
			Presences: c.state.MemberState.RequestPresences,
			Limit:     c.state.MemberState.SearchLimit,
			Nonce:     nonce,
		}); err != nil {
			slog.Error("failed to search guild members", "err", err, "guild_id", gID, "query", name)
		}
		return nil
	}
}

func (c *Model) CacheMemberSearch(event *gateway.GuildMembersChunkEvent) tview.Cmd {
	key, ok := strings.CutPrefix(event.Nonce, memberSearchNonce)
	if !ok {
		return nil
	}

	c.memberSearchCache[key] = uint(len(event.Members))
	return func() tview.Msg {
		return TabSuggestMsg{}
	}
}

func (c *Model) showMentionsList() tview.Cmd {
	c.mentionsVisible = true
	return nil
}

// OnGuildMemberRemove forgets cached member searches that the removed member may have filled.
func (c *Model) OnGuildMemberRemove(event *gateway.GuildMemberRemoveEvent) {
	for name := event.GuildID.String() + " " + event.User.Username; name != ""; name = name[:len(name)-1] {
		if count, ok := c.memberSearchCache[name]; ok && count >= c.state.MemberState.SearchLimit {
			// A full result set may now be missing members; search these prefixes again.
			for name != "" {
				delete(c.memberSearchCache, name)
				name = name[:len(name)-1]
			}
			return
		}
	}
}

// MentionsView places the mentions list just above the composer, near the cursor, or returns nil if it is hidden.
func (c *Model) MentionsView() tview.Element {
	if !c.mentionsVisible {
		return nil
	}
	return mentionsPopup{c}
}

// mentionsPopup lays out the mentions list within the area above the composer, where it is drawn over the messages.
type mentionsPopup struct {
	c *Model
}

func (p mentionsPopup) Draw(screen tview.Screen, area tview.Rectangle) {
	p.c.mentionsList.View().Draw(screen, p.area(area))
}

func (p mentionsPopup) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg {
	return p.c.mentionsList.View().Handle(msg, p.area(area))
}

// area returns where the list goes in area, the messages and composer together.
func (p mentionsPopup) area(area tview.Rectangle) tview.Rectangle {
	c := p.c
	borders := 0
	if c.cfg.Theme.Border.Enabled {
		borders = 1
	}
	x, _ := c.frame()
	maxW, maxH := area.Width, area.Height-c.height
	if t := int(c.cfg.Theme.MentionsList.MaxHeight); t != 0 {
		maxH = min(maxH, t)
	}
	h := min(c.mentionsList.ItemCount()+borders, maxH) + borders + c.cfg.Theme.Border.Padding[1]
	w := int(c.cfg.Theme.MentionsList.MinWidth)
	if w == 0 {
		w = maxW
	} else {
		w = min(max(w, c.mentionsList.MaxDisplayWidth())+borders*2, maxW)
		value, cursor := c.editState.Value(), c.editState.Cursor()
		col := uniseg.StringWidth(value[strings.LastIndex(value[:cursor], "\n")+1 : cursor])
		x += min(col, maxW-w)
	}
	return tview.Rectangle{X: area.X + x, Y: area.Y + area.Height - c.height - h, Width: w, Height: h}
}

func (c *Model) addMentionMember(gID discord.GuildID, m *discord.Member) bool {
	if m == nil {
		return false
	}

	name := m.User.DisplayOrUsername()
	if m.Nick != "" {
		name = m.Nick
	}

	style := tcell.StyleDefault

	// This avoids a slower member color lookup path.
	color, ok := state.MemberColor(m, func(id discord.RoleID) *discord.Role {
		r, _ := c.state.Cabinet.Role(gID, id)
		return r
	})
	if ok {
		style = style.Foreground(tcell.NewHexColor(int32(color)))
	}

	presence, err := c.state.Cabinet.Presence(gID, m.User.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "guild_id", gID, "user_id", m.User.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		style = style.Dim(true)
	}

	c.mentionsList.Append(mentionslist.Item{
		InsertText:  m.User.Username,
		DisplayText: name,
		Style:       style,
	})
	return c.mentionsList.ItemCount() > int(c.cfg.AutocompleteLimit)
}

func (c *Model) addMentionUser(user *discord.User) {
	if user == nil {
		return
	}

	name := user.DisplayOrUsername()
	style := tcell.StyleDefault
	presence, err := c.state.Cabinet.Presence(discord.NullGuildID, user.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "user_id", user.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		style = style.Dim(true)
	}

	c.mentionsList.Append(mentionslist.Item{
		InsertText:  user.Username,
		DisplayText: name,
		Style:       style,
	})
}

func (c *Model) CloseMentions() {
	c.mentionsVisible = false
}

func (c *Model) stopTabCompletion() tview.Cmd {
	if c.cfg.AutocompleteLimit > 0 {
		c.mentionsList.Clear()
		c.CloseMentions()
	}
	return nil
}

func (c *Model) openEditor() tview.Cmd {
	if c.cfg.Editor == "" {
		return func() tview.Msg {
			slog.Warn("Attempt to open file with editor, but no editor is set")
			return nil
		}
	}
	text := c.editState.Value()
	cfg := c.cfg
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

func (c *Model) attach(name string, reader io.Reader) {
	c.sendMessageData.Files = append(c.sendMessageData.Files, sendpart.File{Name: name, Reader: reader})

	var names []string
	for _, file := range c.sendMessageData.Files {
		names = append(names, file.Name)
	}
	c.setFooter("Attached " + humanJoin(names))
}

func (c *Model) canAttachFiles() bool {
	return c.channel != nil && c.state.HasPermissions(c.channel.ID, discord.PermissionAttachFiles)
}

// editAction binds the configured newline key and textarea's other default keys.
func (c Model) editAction(key tview.KeyMsg) (textarea.Action, bool) {
	if keybind.Matches(key, c.cfg.Keybinds.Composer.Newline.Keybind) {
		return textarea.ActionNewline, true
	}
	action, ok := textarea.DefaultKeybind(key)
	return action, ok && action != textarea.ActionNewline
}

func (c Model) ShortHelp() []keybind.Keybind {
	if c.mentionsVisible {
		cfg := c.cfg.Keybinds.MentionsList
		ccfg := c.cfg.Keybinds.Composer
		short := []keybind.Keybind{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, ccfg.TabComplete.Keybind, ccfg.Cancel.Keybind}
		if c.canAttachFiles() {
			short = append(short, ccfg.OpenFilePicker.Keybind)
		}
		return short
	}

	cfg := c.cfg.Keybinds.Composer
	short := []keybind.Keybind{cfg.Send.Keybind, cfg.Newline.Keybind, cfg.Cancel.Keybind, cfg.Paste.Keybind, cfg.OpenEditor.Keybind}
	if c.canEditLastMessage() {
		short = append(short, cfg.EditLast.Keybind)
	}
	if c.sendMessageData.Reference != nil {
		short = append(short, cfg.ToggleReplyMention.Keybind)
	}
	if c.canAttachFiles() {
		short = append(short, cfg.OpenFilePicker.Keybind)
	}
	return short
}

func (c Model) FullHelp() [][]keybind.Keybind {
	if c.mentionsVisible {
		mcfg := c.cfg.Keybinds.MentionsList
		ccfg := c.cfg.Keybinds.Composer
		return [][]keybind.Keybind{
			{mcfg.SelectUp.Keybind, mcfg.SelectDown.Keybind, mcfg.SelectTop.Keybind, mcfg.SelectBottom.Keybind},
			{ccfg.TabComplete.Keybind, ccfg.Cancel.Keybind},
		}
	}

	cfg := c.cfg.Keybinds.Composer
	openEditor := []keybind.Keybind{cfg.Paste.Keybind, cfg.OpenEditor.Keybind}

	if c.canAttachFiles() {
		openEditor = append(openEditor, cfg.OpenFilePicker.Keybind)
	}

	compose := []keybind.Keybind{cfg.Send.Keybind, cfg.Newline.Keybind, cfg.Cancel.Keybind, cfg.Undo.Keybind}
	compose = append(compose, cfg.EditLast.Keybind)
	if c.sendMessageData.Reference != nil {
		compose = append(compose, cfg.ToggleReplyMention.Keybind)
	}

	return [][]keybind.Keybind{
		compose,
		openEditor,
	}
}

func humanJoin(items []string) string {
	count := len(items)
	switch count {
	case 0:
		return ""
	case 1:
		return items[0]
	case 2:
		return items[0] + " and " + items[1]
	default:
		return strings.Join(items[:count-1], ", ") + ", and " + items[count-1]
	}
}
