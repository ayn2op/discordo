package composer

import (
	"bytes"
	"cmp"
	"context"
	"io"
	"log/slog"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/ayn2op/tview/layout"

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
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/textarea"
	"github.com/gdamore/tcell/v3"
	"github.com/rockorager/go-uucode"
	"github.com/yuin/goldmark/v2/ast"
	"golang.design/x/clipboard"
)

const (
	tmpFilePattern      = consts.Name + "_*.md"
	imageAttachmentName = "clipboard.png"
	memberSearchNonce   = "autocomplete:"
)

const (
	noChannelPlaceholder    = "Select a channel to start chatting"
	noPermissionPlaceholder = "You do not have permission to send messages in this channel."
	messagePlaceholder      = "Message..."
)

var mentionRegex = regexp.MustCompile("@[a-zA-Z0-9._]+")

type Model struct {
	text string
	// cursor is the byte offset of the cursor in text, and history what they were before each change, for undo.
	cursor  int
	history []edit

	disabled bool

	state *ningen.State
	// channel is the selected channel, or nil for none.
	channel *discord.Channel

	cfg *config.Config

	editing *discord.Message
	// replyName is the name of the author of the message being replied to.
	replyName         string
	sendMessageData   *api.SendMessageData
	memberSearchCache map[string]uint
	mentionsList      mentionslist.Model
	lastSearch        time.Time

	typingUntil time.Time
}

// TypingDuration is how long Discord shows someone as typing after they start.
const TypingDuration = 10 * time.Second

func NewModel(cfg *config.Config, state *ningen.State) Model {
	return Model{
		disabled:          true,
		cfg:               cfg,
		state:             state,
		sendMessageData:   &api.SendMessageData{},
		memberSearchCache: make(map[string]uint),
		mentionsList:      mentionslist.NewModel(cfg),
	}
}

// SetChannel sets the channel messages are sent to, disabling the composer if the user may not send messages there.
func (m *Model) SetChannel(channel *discord.Channel) {
	m.channel, m.typingUntil = channel, time.Time{}
	isDM := channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM

	// Sending in a thread takes another permission than in the channel it is in.
	sendPerm := discord.PermissionSendMessages
	if channel.Type.IsThread() {
		sendPerm = discord.PermissionSendMessagesInThreads
	}

	m.disabled = !isDM && !m.state.HasPermissions(channel.ID, sendPerm)
}

func (m Model) Disabled() bool {
	return m.disabled
}

func (m Model) placeholder() string {
	switch {
	case m.channel == nil:
		return noChannelPlaceholder
	case m.disabled:
		return noPermissionPlaceholder
	}
	return messagePlaceholder
}

// View shows the text area in a box, taking keys if focused.
func (m Model) View(focused bool) tview.Widget { return m.box(focused) }

// box returns the text area in a box with the title and footer.
func (m *Model) box(focused bool) box.Widget {
	text := textarea.New(m.text, m.cursor).
		Placeholder(m.placeholder()).
		Keybind(m.keybind).
		Focused(focused && !m.disabled).
		OnInput(func(text string, cursor int) tview.Msg { return editMsg{text, cursor} }).
		Height(layout.Shrink.Max(m.cfg.Composer.MaxHeight))
	return ui.Box(text, m.cfg, focused).
		Title(ui.Label(m.title(), m.cfg.Theme.Title, focused)).
		Footer(ui.Label(m.footer(), m.cfg.Theme.Footer, focused)).
		Height(layout.Shrink)
}

// title returns what the composer is doing: editing, replying, or nothing.
func (m Model) title() string {
	switch data := m.sendMessageData; {
	case m.editing != nil:
		return "Editing"
	case data.Reference != nil:
		if mention := data.AllowedMentions.RepliedUser; mention != nil && *mention {
			return "[@] Replying to " + m.replyName
		}
		return "Replying to " + m.replyName
	}
	return ""
}

// footer returns the names of the attached files, or "" if there are none.
func (m Model) footer() string {
	if len(m.sendMessageData.Files) == 0 {
		return ""
	}
	var names []string
	for _, file := range m.sendMessageData.Files {
		names = append(names, file.Name)
	}
	return "Attached " + humanJoin(names)
}

// isBound reports whether key is one of the composer's keybinds.
func (m Model) isBound(key tview.KeyMsg) bool {
	keybinds := m.cfg.Keybinds.Composer
	if keybind.Matches(key,
		keybinds.Paste.Keybind,
		keybinds.Send.Keybind,
		keybinds.OpenEditor.Keybind,
		keybinds.OpenFilePicker.Keybind,
		keybinds.Cancel.Keybind,
		keybinds.TabComplete.Keybind,
		keybinds.ToggleReplyMention.Keybind,
		keybinds.Undo.Keybind,
	) {
		return true
	}
	return keybind.Matches(key, keybinds.EditLast.Keybind) && m.canEditLastMessage()
}

// setText replaces the text and moves the cursor, remembering what they were for undo if the text changed.
func (m *Model) setText(text string, cursor int) {
	if text != m.text {
		m.history = append(m.history, edit{m.text, m.cursor})
	}

	m.text, m.cursor = text, cursor
}

// replace replaces the bytes from start to end of the text with text and moves the cursor after it.
func (m *Model) replace(start, end int, text string) {
	m.setText(m.text[:start]+text+m.text[end:], start+len(text))
}

// insert inserts text at the cursor.
func (m *Model) insert(text string) {
	m.replace(m.cursor, m.cursor, text)
}

// undo restores the text and cursor from before the last change.
func (m *Model) undo() {
	if len(m.history) == 0 {
		return
	}

	last := m.history[len(m.history)-1]
	m.history = m.history[:len(m.history)-1]
	m.text, m.cursor = last.text, last.cursor
}

// pasteText inserts the text on the clipboard at the cursor.
func (m *Model) pasteText() {
	data, err := clipboard.Read(context.Background(), clipboard.FmtText)
	if err != nil {
		slog.Error("failed to read from clipboard", "err", err)
		return
	}
	m.insert(string(data))
}

func (m *Model) reset() {
	m.editing = nil
	m.sendMessageData = &api.SendMessageData{}
	m.setText("", 0)
}

// Update returns m changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case TabSuggestMsg:
		return m, m.tabSuggest()
	case imagePastedMsg:
		if len(msg) == 0 {
			return m, nil
		}
		m.attach(imageAttachmentName, bytes.NewReader(msg))
		return m, nil
	case filesPickedMsg:
		selectedChannel := m.channel
		if selectedChannel == nil || selectedChannel.ID != msg.channelID {
			return m, closeFiles(msg.files)
		}
		for _, file := range msg.files {
			m.attach(file.Name, file.Reader)
		}

		return m, nil
	case editorMsg:
		m.setText(string(msg), len(msg))
		return m, nil
	case editMsg:
		m.setText(msg.text, msg.cursor)
		typingCmd := m.sendTyping()
		if m.cfg.AutocompleteLimit > 0 {
			return m, tview.Batch(typingCmd, m.tabSuggest())
		}
		return m, typingCmd

	case tview.KeyMsg:
		keybinds := m.cfg.Keybinds.Composer
		switch {
		case keybind.Matches(msg, keybinds.EditLast.Keybind) && m.canEditLastMessage():
			return m, editLastMessage()
		case keybind.Matches(msg, keybinds.Paste.Keybind):
			m.pasteText()
			return m, pasteImage()
		case keybind.Matches(msg, keybinds.Send.Keybind):
			if m.mentionsListVisible() {
				return m, m.tabComplete()
			}
			return m, m.send()
		case keybind.Matches(msg, keybinds.OpenEditor.Keybind):
			m.stopTabCompletion()
			return m, m.openEditor()
		case keybind.Matches(msg, keybinds.OpenFilePicker.Keybind):
			m.stopTabCompletion()
			return m, m.pickFiles()
		case keybind.Matches(msg, keybinds.Cancel.Keybind):
			if m.mentionsListVisible() {
				m.stopTabCompletion()
				return m, nil
			}
			m.reset()
			return m, nil
		case keybind.Matches(msg, keybinds.TabComplete.Keybind):
			if m.mentionsListVisible() {
				return m, m.tabComplete()
			}
			m.insert("\t")
			return m, nil
		case keybind.Matches(msg, keybinds.ToggleReplyMention.Keybind):
			m.toggleReplyMention()
			return m, nil
		case keybind.Matches(msg, keybinds.Undo.Keybind):
			m.undo()
			return m, nil
		}
	case mentionslist.Msg:
		var cmd tview.Cmd
		m.mentionsList, cmd = m.mentionsList.Update(msg)
		return m, cmd
	}
	return m, nil
}

func (m Model) canEditLastMessage() bool {
	return !m.Disabled() && m.editing == nil && m.text == "" &&
		m.sendMessageData.Reference == nil && len(m.sendMessageData.Files) == 0 &&
		!m.mentionsListVisible()
}

// StartEdit puts message in the composer to be edited.
func (m *Model) StartEdit(message discord.Message) {
	m.editing = &message
	m.setText(message.Content, len(message.Content))
}

// StartReply makes the next message a reply to message by name, mentioning its author if mention is set.
func (m *Model) StartReply(message discord.Message, name string, mention bool) {
	data := m.sendMessageData
	data.Reference = &discord.MessageReference{MessageID: message.ID}
	data.AllowedMentions = &api.AllowedMentions{RepliedUser: option.Some(mention)}
	m.replyName = name
}

func (m Model) toggleReplyMention() {
	data := m.sendMessageData
	if data.Reference == nil || data.AllowedMentions == nil || data.AllowedMentions.RepliedUser == nil {
		return
	}

	data.AllowedMentions.RepliedUser = option.Some(!*data.AllowedMentions.RepliedUser)
}

func (m Model) processText(channel *discord.Channel, source []byte) string {
	// Fast path: no mentions to expand.
	if bytes.IndexByte(source, '@') == -1 {
		return string(source)
	}

	// Fast path: no back ticks (code blocks), so expand mentions directly.
	if bytes.IndexByte(source, '`') == -1 {
		return string(m.expandMentions(channel, source))
	}

	// Replace back to front so that expanding a mention does not shift the ranges before it.
	for _, textRange := range slices.Backward(mentionableRanges(source)) {
		start, stop := textRange[0], textRange[1]
		source = slices.Replace(source, start, stop, m.expandMentions(channel, source[start:stop])...)
	}

	return string(source)
}

// mentionableRanges returns the ranges of source that are text outside of code, where mentions are expanded.
func mentionableRanges(source []byte) [][2]int {
	var (
		ranges     [][2]int
		canMention = true
	)

	_ = ast.Walk(md.Parse(source), func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		if isCode(node) {
			canMention = !enter
		}

		text, ok := node.(*ast.Text)
		if ok && enter && canMention && !text.Value.IsOwned() {
			index := text.Value.Index()
			ranges = append(ranges, [2]int{index.Start, index.Stop})
		}

		return ast.WalkContinue, nil
	})

	return ranges
}

func isCode(node ast.Node) bool {
	switch node := node.(type) {
	case *ast.CodeBlock:
		return true
	case *md.Inline:
		return node.Attr&md.AttrMonospace != 0
	default:
		return false
	}
}

func (m Model) expandMentions(channel *discord.Channel, source []byte) []byte {
	return mentionRegex.ReplaceAllFunc(source, func(input []byte) []byte {
		if mention := m.mention(channel, string(input[1:])); mention != "" {
			return []byte(mention)
		}

		return input
	})
}

// mention returns the mention of the user called name who can see channel, or "" if there is none.
func (m Model) mention(channel *discord.Channel, name string) string {
	if channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM {
		return m.privateMention(channel, name)
	}

	var mention string
	_ = m.state.MemberStore.Each(channel.GuildID, func(member *discord.Member) bool {
		if !strings.EqualFold(member.User.Username, name) {
			return false
		}

		if channelHasUser(m.state, channel.ID, member.User.ID) {
			mention = member.User.ID.Mention()
		}

		return true
	})

	return mention
}

// privateMention returns the mention of the recipient of channel or the user called name, or "" if neither is.
func (m Model) privateMention(channel *discord.Channel, name string) string {
	for _, user := range channel.DMRecipients {
		if strings.EqualFold(user.Username, name) {
			return user.ID.Mention()
		}
	}

	if me, _ := m.state.Cabinet.Me(); me != nil && strings.EqualFold(me.Username, name) {
		return me.ID.Mention()
	}

	return ""
}

// wordBeforeCursor returns cursor, the runes satisfying isWordRune just before it, and the rune before those.
func wordBeforeCursor(value string, cursor int, isWordRune func(rune) bool) (int, string, rune) {
	for start := cursor; start > 0; {
		previous, size := utf8.DecodeLastRuneInString(value[:start])
		if !isWordRune(previous) {
			return cursor, value[start:cursor], previous
		}

		start -= size
	}

	return cursor, value[:cursor], 0
}

func isMentionChar(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' || char == '.'
}

// suggestRecentAuthors suggests the authors of the channel's recent messages, other than the user.
func (m *Model) suggestRecentAuthors(channel *discord.Channel) {
	messages, err := m.state.Cabinet.Messages(channel.ID)
	if err != nil {
		return
	}
	me, _ := m.state.Cabinet.Me()
	shown := map[discord.UserID]bool{me.ID: true}
	for _, message := range messages {
		if shown[message.Author.ID] {
			continue
		}
		shown[message.Author.ID] = true
		if !channel.GuildID.IsValid() {
			m.addMentionUser(&message.Author)
			continue
		}
		m.state.MemberState.RequestMember(channel.GuildID, message.Author.ID)
		if member, err := m.state.Cabinet.Member(channel.GuildID, message.Author.ID); err == nil && m.addMentionMember(channel.GuildID, member) {
			return
		}
	}
}

type memberList []discord.Member
type userList []discord.User

func (ml memberList) String(index int) string {
	return ml[index].Nick + ml[index].User.DisplayName + ml[index].User.Tag()
}

func (ml memberList) Len() int {
	return len(ml)
}

func (ul userList) String(index int) string {
	return ul[index].DisplayName + ul[index].Tag()
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

func (m Model) CacheMemberSearch(event *gateway.GuildMembersChunkEvent) tview.Cmd {
	key, ok := strings.CutPrefix(event.Nonce, memberSearchNonce)
	if !ok {
		return nil
	}

	m.memberSearchCache[key] = uint(len(event.Members))
	return func() tview.Msg {
		return TabSuggestMsg{}
	}
}

// OnGuildMemberRemove forgets cached member searches that the removed member may have filled.
func (m Model) OnGuildMemberRemove(event *gateway.GuildMemberRemoveEvent) {
	name := event.GuildID.String() + " " + event.User.Username
	for ; name != ""; name = name[:len(name)-1] {
		if count, ok := m.memberSearchCache[name]; ok && count >= m.state.MemberState.SearchLimit {
			break
		}
	}

	// A full result set may now be missing members; search it and its prefixes again.
	for ; name != ""; name = name[:len(name)-1] {
		delete(m.memberSearchCache, name)
	}
}

// MentionsListView places the mentions list at the bottom of the area above the composer, near the cursor, or returns nil if it is empty or the composer is not focused.
func (m Model) MentionsListView(focused bool) tview.Widget {
	if !focused || !m.mentionsListVisible() {
		return nil
	}
	return mentionsPopup{m}
}

// mentionsPopup places the mentions list within the area of the messages, which it is drawn over.
type mentionsPopup struct {
	model Model
}

// Size returns Fill, as the popup places the list within its whole area.
func (mentionsPopup) Size() (width, height layout.Length) { return layout.Fill, layout.Fill }

// Layout returns the size of limits, as the popup places the list within its whole area.
func (mentionsPopup) Layout(limits layout.Limits) layout.Size {
	return layout.Atomic(limits, layout.Fill, layout.Fill)
}

func (p mentionsPopup) Draw(screen tview.Screen, area tview.Rectangle) {
	p.model.mentionsList.View().Draw(screen, p.area(area))
}

func (p mentionsPopup) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg {
	return p.model.mentionsList.View().Handle(msg, p.area(area))
}

// area returns where the list goes in area: at its bottom, at the column of the cursor where it fits.
func (p mentionsPopup) area(area tview.Rectangle) tview.Rectangle {
	model := p.model
	size := model.mentionsList.View().Layout(layout.Limits{Max: layout.Size{Width: area.Width, Height: area.Height}})
	value, cursor := model.text, model.cursor
	column := uucode.StringWidth(value[strings.LastIndex(value[:cursor], "\n")+1 : cursor])
	x := min(model.box(false).InnerArea(tview.Rectangle{}).X+column, area.Width-size.Width)
	return tview.Rectangle{X: area.X + x, Y: area.Y + area.Height - size.Height, Width: size.Width, Height: size.Height}
}

func (m *Model) addMentionMember(guildID discord.GuildID, member *discord.Member) bool {
	if member == nil {
		return false
	}

	name := cmp.Or(member.Nick, member.User.DisplayName, member.User.Username)

	style := tcell.StyleDefault

	// This avoids a slower member color lookup path.
	color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
		role, _ := m.state.Cabinet.Role(guildID, id)
		return role
	})
	if ok {
		style = style.Foreground(tcell.NewHexColor(int32(color)))
	}

	presence, err := m.state.Cabinet.Presence(guildID, member.User.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "guild_id", guildID, "user_id", member.User.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		style = style.Dim(true)
	}

	m.mentionsList.Append(mentionslist.Item{
		InsertText:  member.User.Username,
		DisplayText: name,
		Style:       style,
	})
	return m.mentionsList.ItemCount() > int(m.cfg.AutocompleteLimit)
}

func (m *Model) addMentionUser(user *discord.User) {
	if user == nil {
		return
	}

	name := user.DisplayOrUsername()
	style := tcell.StyleDefault
	presence, err := m.state.Cabinet.Presence(discord.NullGuildID, user.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "user_id", user.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		style = style.Dim(true)
	}

	m.mentionsList.Append(mentionslist.Item{
		InsertText:  user.Username,
		DisplayText: name,
		Style:       style,
	})
}

// stopTabCompletion clears the mentions list, which hides it, when it is in use.
func (m *Model) stopTabCompletion() {
	if m.cfg.AutocompleteLimit > 0 {
		m.mentionsList.Clear()
	}
}

// mentionsListVisible reports whether the mentions list has items to show above the composer.
func (m Model) mentionsListVisible() bool {
	return m.mentionsList.ItemCount() != 0
}

func (m Model) attach(name string, reader io.Reader) {
	m.sendMessageData.Files = append(m.sendMessageData.Files, sendpart.File{Name: name, Reader: reader})
}

func (m Model) canAttachFiles() bool {
	return m.channel != nil && m.state.HasPermissions(m.channel.ID, discord.PermissionAttachFiles)
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
