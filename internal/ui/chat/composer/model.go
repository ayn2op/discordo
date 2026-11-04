package composer

import (
	"bytes"
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
	"github.com/ayn2op/arikawa/v3/utils/json/option"
	"github.com/ayn2op/arikawa/v3/utils/sendpart"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/ui/autocomplete"
	"github.com/ayn2op/discordo/internal/ui/autocomplete/mentions"
	"github.com/ayn2op/discordo/internal/ui/images"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/textarea"
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
	autocomplete      autocomplete.Model
	lastSearch        time.Time

	typingUntil time.Time

	// focused reports whether the composer takes keys and shows its completions.
	focused bool
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
		autocomplete:      autocomplete.NewModel(cfg),
	}
}

// SetChannel sets the channel messages are sent to, disabling the composer if the user may not send messages there.
func (m *Model) SetChannel(channel *discord.Channel) {
	// A reply or an edit is of a message in the channel left; typed text and attachments stay.
	if m.channel != nil && m.channel.ID != channel.ID {
		m.sendMessageData.Reference, m.sendMessageData.AllowedMentions = nil, nil
		if m.editing != nil {
			m.editing = nil
			m.setText("", 0)
		}
	}

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

// SetFocused sets whether the composer takes keys and shows its completions.
func (m *Model) SetFocused(focused bool) {
	m.focused = focused
}

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

func (m Model) canEditLastMessage() bool {
	return !m.Disabled() && m.editing == nil && m.text == "" &&
		m.sendMessageData.Reference == nil && len(m.sendMessageData.Files) == 0 &&
		!m.autocomplete.Visible()
}

// StartEdit puts message in the composer to be edited.
func (m *Model) StartEdit(message discord.Message) {
	m.editing = &message
	m.setText(message.Content, len(message.Content))
}

// CancelEdit stops editing the message with id in the channel, if that is the one being edited, such as once it is deleted.
func (m *Model) CancelEdit(channelID discord.ChannelID, id discord.MessageID) {
	if m.editing == nil || m.editing.ChannelID != channelID || m.editing.ID != id {
		return
	}

	m.reset()
}

// StartReply makes the next message a reply to message by name, mentioning its author if mention is set.
func (m *Model) StartReply(message discord.Message, name string, mention bool) {
	data := m.sendMessageData
	data.Reference = &discord.MessageReference{MessageID: message.ID}
	data.AllowedMentions = &api.AllowedMentions{RepliedUser: option.Some(mention)}
	m.replyName = name
}

func (m *Model) toggleReplyMention() {
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

		if mentions.ChannelHasUser(m.state, channel.ID, member.User.ID) {
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

// wordBeforeCursor returns the runes satisfying isWordRune just before cursor, and the rune before those.
func wordBeforeCursor(value string, cursor int, isWordRune func(rune) bool) (string, rune) {
	for start := cursor; start > 0; {
		previous, size := utf8.DecodeLastRuneInString(value[:start])
		if !isWordRune(previous) {
			return value[start:cursor], previous
		}

		start -= size
	}

	return value[:cursor], 0
}

func isMentionChar(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' || char == '.'
}

func isEmojiChar(char rune) bool {
	return unicode.IsLetter(char) || unicode.IsDigit(char) || char == '_' || char == '+' || char == '-'
}

// query returns the trigger rune and the name being completed, or 0 if none.
func (m Model) query() (rune, string) {
	if name, trigger := wordBeforeCursor(m.text, m.cursor, isMentionChar); trigger == '@' && m.cfg.UI.Autocomplete.Mentions.Enabled {
		return trigger, name
	}

	if !m.cfg.UI.Autocomplete.Emojis.Enabled {
		return 0, ""
	}

	name, trigger := wordBeforeCursor(m.text, m.cursor, isEmojiChar)
	if trigger != ':' {
		return 0, ""
	}

	colon := m.cursor - len(name) - 1
	before, _ := utf8.DecodeLastRuneInString(m.text[:colon])
	startsWord := colon == 0 || unicode.IsSpace(before)
	if !startsWord {
		return 0, ""
	}

	return trigger, name
}

// CacheMemberSearch returns the command that refreshes the autocomplete with the members found.
func (m Model) CacheMemberSearch(event *gateway.GuildMembersChunkEvent) tview.Cmd {
	key, ok := strings.CutPrefix(event.Nonce, memberSearchNonce)
	if !ok {
		return nil
	}

	m.memberSearchCache[key] = uint(len(event.Members))
	return func() tview.Msg {
		return RefreshAutocompleteMsg{}
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

// AutocompleteView returns the autocomplete popup, or nil if it is hidden or the composer is not focused.
func (m Model) AutocompleteView() tview.Widget {
	if !m.focused || !m.autocomplete.Visible() {
		return nil
	}
	return autocompletePopup{m}
}

// autocompletePopup places the autocomplete within the area of the messages, which it is drawn over.
type autocompletePopup struct {
	model Model
}

// Size returns Fill, as the popup places the list within its whole area.
func (autocompletePopup) Size() (width, height layout.Length) { return layout.Fill, layout.Fill }

// Layout returns the size of limits, as the popup places the list within its whole area.
func (autocompletePopup) Layout(limits layout.Limits) layout.Size {
	return layout.Atomic(limits, layout.Fill, layout.Fill)
}

func (p autocompletePopup) Draw(screen tview.Screen, area tview.Rectangle) {
	p.model.autocomplete.View().Draw(screen, p.area(area))
}

func (p autocompletePopup) Handle(msg tview.Msg, area tview.Rectangle) tview.Msg {
	return p.model.autocomplete.View().Handle(msg, p.area(area))
}

// area returns where the list goes in area: at its bottom, at the column of the cursor where it fits.
func (p autocompletePopup) area(area tview.Rectangle) tview.Rectangle {
	model := p.model
	size := model.autocomplete.View().Layout(layout.Limits{Max: layout.Size{Width: area.Width, Height: area.Height}})
	value, cursor := model.text, model.cursor
	column := uucode.StringWidth(value[strings.LastIndex(value[:cursor], "\n")+1 : cursor])
	x := min(model.box(false).InnerArea(tview.Rectangle{}).X+column, area.Width-size.Width)
	return tview.Rectangle{X: area.X + x, Y: area.Y + area.Height - size.Height, Width: size.Width, Height: size.Height}
}

// showCompletions hides the autocomplete if items is empty.
func (m *Model) showCompletions(title string, items []autocomplete.Item) tview.Cmd {
	var cmd tview.Cmd
	m.autocomplete, cmd = m.autocomplete.Update(autocomplete.SetItemsMsg{Title: title, Items: items})
	return cmd
}

func (m *Model) attach(name string, reader io.Reader) {
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

var _ tview.Model[Model] = Model{}

func (m Model) Init() tview.Cmd { return nil }

// Update returns m changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case RefreshAutocompleteMsg:
		return m, m.showAutocomplete()
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
		return m, tview.Batch(typingCmd, m.showAutocomplete())

	case tview.KeyMsg:
		keybinds := m.cfg.Keybinds.Composer
		switch {
		case keybind.Matches(msg, keybinds.EditLast.Keybind) && m.canEditLastMessage():
			return m, editLastMessage()
		case keybind.Matches(msg, keybinds.Paste.Keybind):
			m.pasteText()
			return m, pasteImage()
		case keybind.Matches(msg, keybinds.Send.Keybind):
			if m.autocomplete.Visible() {
				return m, m.complete()
			}
			return m, m.send()
		case keybind.Matches(msg, keybinds.OpenEditor.Keybind):
			return m, tview.Batch(m.showCompletions("", nil), m.openEditor())
		case keybind.Matches(msg, keybinds.OpenFilePicker.Keybind):
			return m, tview.Batch(m.showCompletions("", nil), m.pickFiles())
		case keybind.Matches(msg, keybinds.Cancel.Keybind):
			if m.autocomplete.Visible() {
				return m, m.showCompletions("", nil)
			}
			m.reset()
			return m, nil
		case keybind.Matches(msg, keybinds.TabComplete.Keybind):
			if m.autocomplete.Visible() {
				return m, m.complete()
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
	case autocomplete.Msg, images.ChangedMsg:
		var cmd tview.Cmd
		m.autocomplete, cmd = m.autocomplete.Update(msg)
		return m, cmd
	}
	return m, nil
}

// View shows the text area in a box, taking keys if focused.
func (m Model) View() tview.Widget { return m.box(m.focused) }
