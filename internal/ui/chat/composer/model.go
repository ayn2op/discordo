package composer

import (
	"bytes"
	"cmp"
	"context"
	"github.com/ayn2op/tview/layout"
	"io"
	"log/slog"
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
	"github.com/rivo/uniseg"
	"github.com/yuin/goldmark/ast"
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
	editState textarea.EditState
	disabled  bool

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

	// mentionsVisible reports whether the mentions list is shown above the composer.
	mentionsVisible bool

	typingUntil time.Time
}

// TypingDuration is how long Discord shows someone as typing after they start.
const TypingDuration = 10 * time.Second

var _ help.KeyMap = Model{}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	c := Model{
		disabled:          true,
		cfg:               cfg,
		state:             state,
		sendMessageData:   &api.SendMessageData{},
		memberSearchCache: make(map[string]uint),
		mentionsList:      mentionslist.NewModel(cfg),
	}
	return c
}

// SetChannel sets the channel messages are sent to, disabling the composer if the user may not send messages there.
func (m *Model) SetChannel(channel *discord.Channel) {
	m.channel, m.typingUntil = channel, time.Time{}
	isDM := channel.Type == discord.DirectMessage || channel.Type == discord.GroupDM
	m.disabled = !isDM && !m.state.HasPermissions(channel.ID, discord.PermissionSendMessages)
}

// Height returns the number of rows the composer takes, including its border.
func (m *Model) Height() int {
	_, frame := m.frame()
	return min(strings.Count(m.editState.Value(), "\n")+1, max(m.cfg.Composer.MaxHeight, 1)) + frame
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
func (m Model) View(focused bool) tview.Widget { return view{&m, focused} }

// view is the composer as shown, with or without the focus.
type view struct {
	*Model
	focused bool
}

func (v view) Size() (width, height layout.Length) {
	return v.box(v.focused).Size()
}

func (v view) Layout(limits layout.Limits) layout.Size {
	return v.box(v.focused).Layout(limits)
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
func (m *Model) box(focused bool) box.Widget {
	text := textarea.New(&m.editState).
		Placeholder(m.placeholder()).
		Keybind(m.editAction).
		Focused(focused && !m.disabled).
		OnChange(func(a textarea.Change) tview.Msg { return editMsg(a) })
	return ui.Box(text, &m.cfg.Theme, focused).Title(m.title()).Footer(m.footer())
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
	k := m.cfg.Keybinds.Composer
	if keybind.Matches(key, k.Paste.Keybind, k.Send.Keybind, k.OpenEditor.Keybind, k.OpenFilePicker.Keybind, k.Cancel.Keybind, k.TabComplete.Keybind, k.ToggleReplyMention.Keybind, k.Undo.Keybind) {
		return true
	}
	return keybind.Matches(key, k.EditLast.Keybind) && m.canEditLastMessage()
}

// insert inserts text at the cursor.
func (m *Model) insert(text string) {
	m.editState.Replace(m.editState.Cursor(), m.editState.Cursor(), text)
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

// frame returns the cells the composer's border and padding take on each side.
func (m *Model) frame() (left, vertical int) {
	const size = 100
	inner := m.box(false).InnerArea(tview.Rectangle{Width: size, Height: size})
	return inner.X, size - inner.Height
}

func (m *Model) reset() {
	m.editing = nil
	m.sendMessageData = &api.SendMessageData{}
	m.editState.SetValue("")
}

// Update returns c changed in response to msg and a command to run, or nil.
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
		m.editState.SetValue(string(msg))
		return m, nil
	case editMsg:
		m.editState.Apply(textarea.Change(msg))
		typingCmd := m.sendTyping()
		if m.cfg.AutocompleteLimit > 0 {
			return m, tview.Batch(typingCmd, m.tabSuggest())
		}
		return m, typingCmd

	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.EditLast.Keybind) && m.canEditLastMessage():
			return m, m.editLastMessage()
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.Paste.Keybind):
			m.pasteText()
			return m, pasteImage()
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.Send.Keybind):
			if m.mentionsVisible {
				return m, m.tabComplete()
			}
			return m, m.send()
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.OpenEditor.Keybind):
			m.stopTabCompletion()
			return m, m.openEditor()
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.OpenFilePicker.Keybind):
			m.stopTabCompletion()
			return m, m.pickFiles()
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.Cancel.Keybind):
			if m.mentionsVisible {
				m.stopTabCompletion()
				return m, nil
			}
			m.reset()
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.TabComplete.Keybind):
			if m.mentionsVisible {
				return m, m.tabComplete()
			}
			m.insert("\t")
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.ToggleReplyMention.Keybind):
			m.toggleReplyMention()
			return m, nil
		case keybind.Matches(msg, m.cfg.Keybinds.Composer.Undo.Keybind):
			m.editState.Undo()
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
	return !m.Disabled() && m.editing == nil && m.editState.Value() == "" &&
		m.sendMessageData.Reference == nil && len(m.sendMessageData.Files) == 0 &&
		!m.mentionsVisible
}

// StartEdit puts message in the composer to be edited.
func (m *Model) StartEdit(message discord.Message) {
	m.editing = &message
	m.editState.SetValue(message.Content)
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

func (m Model) processText(channel *discord.Channel, src []byte) string {
	// Fast path: no mentions to expand.
	if bytes.IndexByte(src, '@') == -1 {
		return string(src)
	}

	// Fast path: no back ticks (code blocks), so expand mentions directly.
	if bytes.IndexByte(src, '`') == -1 {
		return string(m.expandMentions(channel, src))
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
			if enter && canMention {
				ranges = append(ranges, [2]int{node.Segment.Start, node.Segment.Stop})
			}
		}
		return ast.WalkContinue, nil
	})

	// Replace back to front so that expanding a mention does not shift the ranges before it.
	for _, rng := range slices.Backward(ranges) {
		src = slices.Replace(src, rng[0], rng[1], m.expandMentions(channel, src[rng[0]:rng[1]])...)
	}

	return string(src)
}

func (m Model) expandMentions(channel *discord.Channel, src []byte) []byte {
	state := m.state
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
	for name := event.GuildID.String() + " " + event.User.Username; name != ""; name = name[:len(name)-1] {
		if count, ok := m.memberSearchCache[name]; ok && count >= m.state.MemberState.SearchLimit {
			// A full result set may now be missing members; search these prefixes again.
			for name != "" {
				delete(m.memberSearchCache, name)
				name = name[:len(name)-1]
			}
			return
		}
	}
}

// MentionsView places the mentions list just above the composer, near the cursor, or returns nil if it is hidden.
func (m *Model) MentionsView() tview.Widget {
	if !m.mentionsVisible {
		return nil
	}
	return mentionsPopup{m}
}

// mentionsPopup lays out the mentions list within the area above the composer, where it is drawn over the messages.
type mentionsPopup struct {
	c *Model
}

// Size returns Fill, as the popup places the list within its whole area.
func (mentionsPopup) Size() (width, height layout.Length) { return layout.Fill, layout.Fill }

// Layout returns the size of limits, as the popup places the list within its whole area.
func (mentionsPopup) Layout(limits layout.Limits) layout.Size {
	return layout.Atomic(limits, layout.Fill, layout.Fill)
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
	maxW, maxH := area.Width, area.Height-c.Height()
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
	return tview.Rectangle{X: area.X + x, Y: area.Y + area.Height - c.Height() - h, Width: w, Height: h}
}

func (m *Model) addMentionMember(gID discord.GuildID, member *discord.Member) bool {
	if member == nil {
		return false
	}

	name := cmp.Or(member.Nick, member.User.DisplayName, member.User.Username)

	style := tcell.StyleDefault

	// This avoids a slower member color lookup path.
	color, ok := state.MemberColor(member, func(id discord.RoleID) *discord.Role {
		r, _ := m.state.Cabinet.Role(gID, id)
		return r
	})
	if ok {
		style = style.Foreground(tcell.NewHexColor(int32(color)))
	}

	presence, err := m.state.Cabinet.Presence(gID, member.User.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "guild_id", gID, "user_id", member.User.ID, "err", err)
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

// stopTabCompletion clears and closes the mentions list when it is in use.
func (m *Model) stopTabCompletion() {
	if m.cfg.AutocompleteLimit > 0 {
		m.mentionsList.Clear()
		m.CloseMentions()
	}
}

func (m *Model) CloseMentions() {
	m.mentionsVisible = false
}

func (m Model) attach(name string, reader io.Reader) {
	m.sendMessageData.Files = append(m.sendMessageData.Files, sendpart.File{Name: name, Reader: reader})
}

func (m Model) canAttachFiles() bool {
	return m.channel != nil && m.state.HasPermissions(m.channel.ID, discord.PermissionAttachFiles)
}

// editAction binds the configured newline key and textarea's other default keys.
func (m Model) editAction(key tview.KeyMsg) textarea.Action {
	if keybind.Matches(key, m.cfg.Keybinds.Composer.Newline.Keybind) {
		return textarea.ActionNewline
	}
	if action := textarea.DefaultKeybind(key); action != textarea.ActionNewline {
		return action
	}
	return textarea.ActionNone
}

func (m Model) ShortHelp() []keybind.Keybind {
	if m.mentionsVisible {
		cfg := m.cfg.Keybinds.MentionsList
		ccfg := m.cfg.Keybinds.Composer
		short := []keybind.Keybind{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, ccfg.TabComplete.Keybind, ccfg.Cancel.Keybind}
		if m.canAttachFiles() {
			short = append(short, ccfg.OpenFilePicker.Keybind)
		}
		return short
	}

	cfg := m.cfg.Keybinds.Composer
	short := []keybind.Keybind{cfg.Send.Keybind, cfg.Newline.Keybind, cfg.Cancel.Keybind, cfg.Paste.Keybind, cfg.OpenEditor.Keybind}
	if m.canEditLastMessage() {
		short = append(short, cfg.EditLast.Keybind)
	}
	if m.sendMessageData.Reference != nil {
		short = append(short, cfg.ToggleReplyMention.Keybind)
	}
	if m.canAttachFiles() {
		short = append(short, cfg.OpenFilePicker.Keybind)
	}
	return short
}

func (m Model) FullHelp() [][]keybind.Keybind {
	if m.mentionsVisible {
		mcfg := m.cfg.Keybinds.MentionsList
		ccfg := m.cfg.Keybinds.Composer
		return [][]keybind.Keybind{
			{mcfg.SelectUp.Keybind, mcfg.SelectDown.Keybind, mcfg.SelectTop.Keybind, mcfg.SelectBottom.Keybind},
			{ccfg.TabComplete.Keybind, ccfg.Cancel.Keybind},
		}
	}

	cfg := m.cfg.Keybinds.Composer
	openEditor := []keybind.Keybind{cfg.Paste.Keybind, cfg.OpenEditor.Keybind}

	if m.canAttachFiles() {
		openEditor = append(openEditor, cfg.OpenFilePicker.Keybind)
	}

	compose := []keybind.Keybind{cfg.Send.Keybind, cfg.Newline.Keybind, cfg.Cancel.Keybind, cfg.Undo.Keybind}
	compose = append(compose, cfg.EditLast.Keybind)
	if m.sendMessageData.Reference != nil {
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
