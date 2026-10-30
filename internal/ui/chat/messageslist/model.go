package messageslist

import (
	"slices"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/discordo/internal/voice"
	"github.com/ayn2op/ningen/v3"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/box"
	"github.com/ayn2op/tview/column"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/layout"
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/row"
	"github.com/ayn2op/tview/scrollbar"
	"github.com/ayn2op/tview/spacer"
	"github.com/ayn2op/tview/text"
	"github.com/ayn2op/tview/viewport"
	"github.com/yuin/goldmark/v2/ast"
)

// placeholderTerminals are the terminals, by XTVERSION name, that draw kitty's Unicode placeholders.
// WezTerm and Konsole support kitty graphics but not Unicode placeholders.
var placeholderTerminals = []string{"kitty", "ghostty", "Rio"}

type messageItem struct {
	message discord.Message
	// view is the message as drawn, keeping its wrapped text, or nil until it is shown and after what it shows changes.
	view tview.Widget
	// separator is the line with the date of the message after it, or nil for a message.
	separator tview.Widget
	// grouped reports whether the message is drawn without a header, under that of the message before it.
	grouped bool

	// A preview is blank while loading and empty if loading failed.
	previews map[discord.URL]tviewimage.Widget
	// root is the parsed markdown of the message's content, or nil until it is needed and after the text is invalidated, as it resolves mentions when parsed.
	root ast.Node
}

type Model struct {
	// cursor is the selected item, or -1 for none, and target the item selected last, which stays in view once deselected.
	cursor, target int
	scrollState    viewport.ScrollState

	cfg   *config.Config
	state *ningen.State
	// channel is the selected channel, or nil for none.
	channel *discord.Channel
	items   []messageItem

	renderer *markdown.Renderer
	// images are the avatars and emoji, shared by every message that shows them.
	images map[discord.URL]tviewimage.Widget
	// voicePlayer plays the voice message played last, and voiceStatus is what it last said of it.
	voicePlayer *voice.Player
	voiceStatus voice.PlayerStatus
	// seeking reports whether the waveform of the voice message is dragged, which is sought once it is let go of.
	seeking bool

	// kitty reports whether previews are drawn with kitty's graphics protocol.
	// kittyImages counts the images sent to the terminal.
	kitty       bool
	kittyImages int
	// cellWidth and cellHeight are the size of a terminal cell in pixels, or zero if the terminal does not report it.
	cellWidth, cellHeight uint
}

func NewModel(cfg *config.Config, state *ningen.State) Model {
	ml := Model{
		cursor:      -1,
		target:      -1,
		cfg:         cfg,
		state:       state,
		renderer:    markdown.NewRenderer(cfg),
		images:      map[discord.URL]tviewimage.Widget{},
		voicePlayer: voice.NewPlayer(),
	}

	ml.scrollState.ScrollToEnd()
	return ml
}

// View shows the messages in a box titled with the channel and footed with footer, such as who is typing.
func (m Model) View(focused bool, footer string) tview.Widget {
	return ui.Box(m.listView(focused), m.cfg, focused).
		Title(ui.Label(m.title(), m.cfg.Theme.Title, focused)).
		Footer(ui.Label(footer, m.cfg.Theme.Footer, focused))
}

// title returns the selected channel and its topic, or "Messages" when none is selected.
func (m Model) title() string {
	if m.channel == nil {
		return "Messages"
	}
	title := ui.ChannelToString(*m.channel, m.cfg.Icons, m.state)
	if topic := m.channel.Topic; topic != "" {
		title += " - " + topic
	}
	return title
}

func (m Model) listView(focused bool) viewport.Widget {
	cfg, theme := m.cfg.UI.ScrollBar, m.cfg.Theme.ScrollBar
	bar := scrollbar.New().
		SymbolSet(cfg.SymbolSet.SymbolSet).
		Style(theme.TrackStyle.Style).
		ThumbStyle(theme.ThumbStyle.Style)
	// The model moves the selection itself, so the list is not focused and only takes clicks.
	l := list.New(m.cursor, len(m.items), m.buildItem).
		// A group's messages have no spacing between them.
		Spacing(func(index int) int {
			if m.items[index].grouped {
				return 0
			}
			return int(m.cfg.UI.MessagesList.Spacing)
		}).
		SelectedStyle(m.cfg.Theme.MessagesList.SelectedMessageStyle.Style).
		OnChange(func(cursor int) tview.Msg { return listMsg(cursor) })
	return viewport.New(l, m.scrollState).
		Target(l.Target(m.target)).
		ScrollBar(bar, cfg.Visibility.ScrollBarVisibility).
		// Older messages are added above, so the position is kept from the newest, which also keeps the newest in view.
		AnchorBottom().
		Keybind(ui.ScrollKeybind(m.cfg.Keybinds.MessagesList.ScrollKeybinds)).
		Focused(focused).
		OnChange(func(s *viewport.ScrollState) tview.Msg { return scrollMsg(*s) })
}

// setCursor selects the item at index and scrolls to it if it changed, or selects none if it is negative.
func (m *Model) setCursor(index int) {
	if index >= 0 && index != m.cursor {
		m.scrollState.ScrollToTarget()
		m.target = index
	}

	m.cursor = max(index, -1)
}

// SetChannel shows messages of channel, newest first as Discord sends them, scrolled to the newest, and requests the members who wrote them.
func (m *Model) SetChannel(channel *discord.Channel, messages []discord.Message) tview.Cmd {
	m.channel = channel
	m.reset()
	m.setMessages(messages)
	if channel.GuildID.IsValid() {
		return tview.Batch(m.requestGuildMembers(channel.GuildID, messages), m.loadPreviews())
	}
	return m.loadPreviews()
}

// ShowNewest clears the selection and scrolls to the newest message, which then stays in view as messages arrive until the user scrolls up.
func (m *Model) ShowNewest() {
	m.clearSelection()
	m.scrollState.ScrollToEnd()
}

// UpdateMessage replaces the shown message with the same ID.
func (m *Model) UpdateMessage(message discord.Message) tview.Cmd {
	m.setMessage(m.indexOf(message.ChannelID, message.ID), message)
	return m.loadPreviews()
}

// RefreshMessage shows the cached state of a message, such as after a reaction.
func (m *Model) RefreshMessage(channelID discord.ChannelID, id discord.MessageID) tview.Cmd {
	message, err := m.state.Cabinet.Message(channelID, id)
	if err != nil {
		return nil
	}

	m.setMessage(m.indexOf(channelID, id), *message)
	return m.loadPreviews()
}

// DeleteMessage removes the shown message with id.
func (m *Model) DeleteMessage(channelID discord.ChannelID, id discord.MessageID) {
	m.deleteMessage(m.indexOf(channelID, id))
}

// indexOf returns the index of the message with id in the channel, or -1 if it is not shown.
func (m Model) indexOf(channelID discord.ChannelID, id discord.MessageID) int {
	if m.channel == nil || m.channel.ID != channelID {
		return -1
	}
	return slices.IndexFunc(m.items, func(item messageItem) bool { return item.separator == nil && item.message.ID == id })
}

func (m *Model) reset() {
	m.items = nil
	clear(m.images)
	m.cursor, m.target = -1, -1
	m.scrollState = viewport.ScrollState{}
	m.scrollState.ScrollToEnd()
}

func (m *Model) setMessages(messages []discord.Message) {
	m.items = make([]messageItem, 0, len(messages))
	for _, message := range slices.Backward(messages) {
		m.items = append(m.items, messageItem{message: message})
	}
	m.rebuildItems()
}

func (m *Model) AddMessage(message discord.Message) tview.Cmd {
	// Appending keeps the items before it as they are, so only the separator of a new day is added rather than all rebuilt.
	var previous *discord.Message
	if last := len(m.items) - 1; last >= 0 {
		previous = &m.items[last].message
	}

	separator := m.shouldSeparate(previous, message)
	if separator {
		m.items = append(m.items, messageItem{separator: m.separatorView(message)})
	}

	grouped := !separator && m.shouldGroup(previous, message)
	m.items = append(m.items, messageItem{message: message, grouped: grouped})
	return m.loadPreviews()
}

func (m *Model) setMessage(index int, message discord.Message) {
	if index < 0 || index >= len(m.items) || m.items[index].separator != nil {
		return
	}

	m.items[index] = messageItem{message: message, previews: m.items[index].previews}
	m.rebuildItems()
}

func (m *Model) deleteMessage(index int) {
	if index < 0 || index >= len(m.items) || m.items[index].separator != nil {
		return
	}

	cursor := m.cursor
	if cursor == index {
		cursor = m.messageIndex(index, -1)
		if cursor == -1 {
			cursor = m.messageIndex(index, 1)
		}
	}
	if cursor > index {
		cursor--
	}
	m.items = slices.Delete(m.items, index, index+1)
	m.setCursor(cursor)
	m.rebuildItems()
}

func (m *Model) clearSelection() {
	m.setCursor(-1)
}

func (m Model) buildItem(index int) tview.Widget {
	if index < 0 || index >= len(m.items) {
		return nil
	}
	item := &m.items[index]
	if item.separator != nil {
		return item.separator
	}

	// A voice message shows its player, which changes as it plays.
	if item.view == nil || isVoiceMessage(item.message) {
		item.view = m.itemView(item)
	}
	return item.view
}

// itemView returns the message of item with its header or avatar, embeds, previews, reactions and what it replies to.
func (m Model) itemView(item *messageItem) tview.Widget {
	message := &item.message
	root := m.contentRoot(item)
	style := m.cfg.Theme.MessagesList.MessageStyle.Style

	// Attachments are named in the text, before the embeds.
	body := column.New(messageText(m.renderMessage(*message, root, style, item.grouped))).Height(layout.Shrink)
	for _, embed := range m.embedViews(*message, root) {
		body = body.Push(embed)
	}
	if isVoiceMessage(*message) {
		body = body.Push(m.voiceMessageView(message.Attachments[0]))
	}

	for _, s := range m.previewSources(*message) {
		if preview, ok := item.previews[s.proxy]; ok {
			body = body.Push(preview)
		}
	}

	if !m.blocked(message.Author) {
		body = body.Push(m.reactionsView(message.Reactions))
	}

	var view tview.Widget = body
	if avatar, ok := m.images[avatarURL(message.Author)]; ok {
		var gutter tview.Widget = avatar
		// A grouped message leaves the avatar's column empty, under the avatar of the group.
		if item.grouped {
			width, _ := avatar.Size()
			gutter = spacer.New().Width(width)
		}
		view = row.New(gutter, body).Spacing(1).Height(layout.Shrink)
	}

	if message.Type == discord.InlinedReplyMessage && !m.blocked(message.Author) {
		builder := new(markdown.Builder)
		m.drawReply(builder, *message, style)
		view = column.New(messageText(builder.Finish()), view).Height(layout.Shrink)
	}

	return view
}

// shouldGroup reports whether message is drawn under the header of previous, the message before it, which is nil for the first.
func (m Model) shouldGroup(previous *discord.Message, message discord.Message) bool {
	cfg := m.cfg.UI.MessagesList.Grouping
	return cfg.Enabled && previous != nil &&
		// Replies and system messages start a group, as their header says more than who wrote them.
		message.Type == discord.DefaultMessage &&
		(previous.Type == discord.DefaultMessage || previous.Type == discord.InlinedReplyMessage) &&
		// Webhooks post under one ID with a name of each message's own.
		message.Author.ID == previous.Author.ID && message.Author.Username == previous.Author.Username &&
		message.Timestamp.Time().Sub(previous.Timestamp.Time()) <= cfg.Interval
}

// separatorDatePadding is on either side of the date on its separator line.
const separatorDatePadding = " "

// separatorView returns the line with the date of message in the middle of it. It is made once, when the separator is added, as the list builds every item each time it is laid out.
func (m Model) separatorView(message discord.Message) tview.Widget {
	date := message.Timestamp.Time().In(time.Local).Format(m.cfg.DateSeparator.Format)
	style := m.cfg.Theme.MessagesList.MessageStyle.Dim(true)
	return box.New(spacer.New()).
		Borders(tview.BordersTop).
		BorderSet(tview.BorderSet{Top: m.cfg.DateSeparator.Character}).
		BorderStyle(style).
		// The separator is kept, so its date is laid out once.
		Title(text.New(separatorDatePadding + date + separatorDatePadding).Style(style).Wrapping(text.WrappingNone).Paragraph(new(text.Paragraph))).
		Height(layout.Fixed(1))
}

// rebuildItems replaces date separators and groups while retaining message views and selection.
func (m *Model) rebuildItems() {
	items := make([]messageItem, 0, len(m.items))
	cursor, target := -1, -1
	var previous *discord.Message
	for i, item := range m.items {
		if item.separator != nil {
			continue
		}

		separator := m.shouldSeparate(previous, item.message)
		if separator {
			items = append(items, messageItem{separator: m.separatorView(item.message)})
		}
		// A message that joins or leaves a group is drawn again, with or without its header.
		if grouped := !separator && m.shouldGroup(previous, item.message); grouped != item.grouped {
			item.grouped, item.view = grouped, nil
		}
		if i == m.cursor {
			cursor = len(items)
		}
		if i == m.target {
			target = len(items)
		}
		items = append(items, item)
		previous = &item.message
	}
	m.items = items
	// The selected and target messages only move with the separators, so they are not scrolled to.
	m.cursor, m.target = cursor, target
}

// shouldSeparate reports whether a date separator goes before message, as it is the first (previous is nil) or starts a new day.
func (m Model) shouldSeparate(previous *discord.Message, message discord.Message) bool {
	return m.cfg.DateSeparator.Enabled && (previous == nil || !sameLocalDate(previous.Timestamp, message.Timestamp))
}

func sameLocalDate(a discord.Timestamp, b discord.Timestamp) bool {
	ta := a.Time().In(time.Local)
	tb := b.Time().In(time.Local)
	return ta.Year() == tb.Year() && ta.YearDay() == tb.YearDay()
}

// messageIndex finds the next message in direction (-1 or 1), excluding start.
func (m Model) messageIndex(start, direction int) int {
	for i := start + direction; i >= 0 && i < len(m.items); i += direction {
		if m.items[i].separator == nil {
			return i
		}
	}
	return -1
}

func (m *Model) onRowCursorChanged(index int) {
	if index < 0 || index >= len(m.items) || m.items[index].separator == nil {
		return
	}
	target := m.messageIndex(index, -1)
	if target == -1 {
		target = m.messageIndex(index, 1)
	}
	m.setCursor(target)
}

func (m Model) selectedMessage() (*discord.Message, bool) {
	cursor := m.cursor
	if cursor < 0 || cursor >= len(m.items) || m.items[cursor].separator != nil {
		return nil, false
	}

	return &m.items[cursor].message, true
}

// selectedURLs returns the web links of the selected message, from its saved markdown.
func (m Model) selectedURLs() []string {
	if message, ok := m.selectedMessage(); ok {
		return m.messageURLs(*message, m.contentRoot(&m.items[m.cursor]))
	}
	return nil
}

// Update returns ml changed in response to msg and a command to run, or nil.
func (m Model) Update(msg tview.Msg) (Model, tview.Cmd) {
	switch msg := msg.(type) {
	case tview.KeyMsg:
		kbs := m.cfg.Keybinds.MessagesList
		switch {
		case keybind.Matches(msg, kbs.Cancel.Keybind):
			m.clearSelection()
		case keybind.Matches(msg, kbs.SelectUp.Keybind):
			return m, m.selectUp()
		case keybind.Matches(msg, kbs.SelectDown.Keybind):
			m.selectDown()
		case keybind.Matches(msg, kbs.SelectTop.Keybind):
			return m, m.selectTop()
		case keybind.Matches(msg, kbs.SelectBottom.Keybind):
			m.selectBottom()
		case keybind.Matches(msg, kbs.SelectReply.Keybind):
			m.selectReply()
		case keybind.Matches(msg, kbs.YankID.Keybind):
			return m, m.yankMessageID()
		case keybind.Matches(msg, kbs.YankContent.Keybind):
			return m, m.yankContent()
		case keybind.Matches(msg, kbs.YankURL.Keybind):
			return m, m.yankURL()
		case keybind.Matches(msg, kbs.Open.Keybind):
			return m, m.open()
		case keybind.Matches(msg, kbs.OpenInBrowser.Keybind):
			return m, m.openInBrowser()
		case keybind.Matches(msg, kbs.Play.Keybind):
			if message, ok := m.selectedMessage(); ok && isVoiceMessage(*message) {
				return m, m.play(message.Attachments[0])
			}
		case keybind.Matches(msg, kbs.SeekBackward.Keybind):
			return m, m.seek(m.voiceStatus.Position - seekStep)
		case keybind.Matches(msg, kbs.SeekForward.Keybind):
			return m, m.seek(m.voiceStatus.Position + seekStep)
		case keybind.Matches(msg, kbs.Download.Keybind):
			return m, m.download()
		case keybind.Matches(msg, kbs.Remind.Keybind):
			return m, m.remind()
		case keybind.Matches(msg, kbs.Reply.Keybind):
			return m, m.reply(false)
		case keybind.Matches(msg, kbs.ReplyMention.Keybind):
			return m, m.reply(true)
		case keybind.Matches(msg, kbs.Edit.Keybind):
			return m, m.editSelectedMessage()
		case keybind.Matches(msg, kbs.Delete.Keybind):
			return m, m.deleteSelectedMessage()
		case keybind.Matches(msg, kbs.DeleteConfirm.Keybind):
			return m, m.confirmDelete()
		}
	case olderMessagesLoadedMsg:
		selectedChannel := m.channel
		if selectedChannel == nil || selectedChannel.ID != msg.channelID {
			return m, nil
		}
		older := make([]messageItem, len(msg.older))
		for i, message := range msg.older {
			older[i] = messageItem{message: message}
		}
		m.items = slices.Concat(older, m.items)
		// The selected and target messages stay the same, so they are not scrolled to.
		if m.cursor >= 0 {
			m.cursor += len(older)
		}
		if m.target >= 0 {
			m.target += len(older)
		}
		m.rebuildItems()
		if selectedChannel.GuildID.IsValid() {
			return m, tview.Batch(m.requestGuildMembers(selectedChannel.GuildID, msg.older), m.loadPreviews())
		}
		return m, m.loadPreviews()
	case previewLoadedMsg:
		preview, cmd := msg.preview, tview.Cmd(nil)
		if m.kitty {
			m.kittyImages++
			preview = preview.Kitty(m.kittyImages%63488 + 1)
			cmd = preview.Transmit()
		}

		if _, ok := m.images[msg.proxy]; ok {
			m.images[msg.proxy] = preview
			// An emoji is drawn within the text of the messages that use it, so the text is rendered again.
			for i := range m.items {
				m.items[i].view = nil
			}
			return m, cmd
		}

		index := slices.IndexFunc(m.items, func(item messageItem) bool { return item.separator == nil && item.message.ID == msg.messageID })
		if index < 0 || m.items[index].previews == nil {
			return m, nil
		}

		m.items[index].previews[msg.proxy] = preview
		m.items[index].view = nil
		return m, cmd
	case playMsg:
		return m, m.play(discord.Attachment(msg))
	case seekPressMsg:
		m.seeking = true
		m.voiceStatus.Position = time.Duration(msg)
		return m, nil
	case seekMsg:
		m.voiceStatus.Position = time.Duration(msg)
		if m.seeking {
			return m, nil
		}

		return m, m.seek(m.voiceStatus.Position)
	case seekReleaseMsg:
		m.seeking = false
		return m, m.seek(m.voiceStatus.Position)
	case voiceMsg:
		status := m.voicePlayer.Status()
		// The waveform shows where it is dragged to rather than what is playing, while that voice message is the one loaded.
		m.seeking = m.seeking && status.ID == m.voiceStatus.ID
		if m.seeking {
			status.Position = m.voiceStatus.Position
		}

		m.voiceStatus = status
		return m, m.Init()
	case tview.ResizeMsg:
		cols, rows := msg.Size()
		width, height := msg.PixelSize()
		if cols > 0 && rows > 0 {
			m.cellWidth, m.cellHeight = uint(width/cols), uint(height/rows)
		}
	case tview.TerminalInfoMsg:
		switch m.cfg.UI.MessagesList.ImageProtocol {
		case "kitty":
			m.kitty = true
		case "halfblocks":
			m.kitty = false
		default:
			m.kitty = slices.Contains(placeholderTerminals, msg.Name)
		}
	case scrollMsg:
		m.scrollState = viewport.ScrollState(msg)
		// Older messages are fetched once as the list is scrolled to the top, rather than on each scroll there.
		if !scrolledToTop(m.scrollState.Origin()) && scrolledToTop(m.scrollState.Viewport()) {
			return m, m.fetchOlderMessages()
		}
	case listMsg:
		top := m.atTop()
		m.cursor, m.target = int(msg), int(msg)
		m.onRowCursorChanged(m.cursor)
		if !top && m.atTop() {
			return m, m.fetchOlderMessages()
		}
	}
	return m, nil
}

// scrolledToTop reports whether v is scrolled to the oldest message shown, which is false while the messages fit, as there is nothing to scroll.
func scrolledToTop(v viewport.Viewport) bool {
	return v.Content().Height > v.Bounds().Height && v.AbsoluteOffsetReversed().Y == 0
}

// atTop reports whether the oldest message shown is selected.
func (m Model) atTop() bool {
	return m.cursor >= 0 && m.cursor == m.messageIndex(-1, 1)
}

func (m *Model) selectDown() {
	if m.cursor == -1 {
		m.selectBottom()
	} else if next := m.messageIndex(m.cursor, 1); next >= 0 {
		m.setCursor(next)
	}
}

func (m *Model) selectTop() tview.Cmd {
	if m.atTop() {
		return nil
	}
	m.setCursor(m.messageIndex(-1, 1))
	return m.fetchOlderMessages()
}

func (m *Model) selectBottom() {
	m.setCursor(m.messageIndex(len(m.items), -1))
}

func (m *Model) selectReply() {
	messages := m.items
	cursor := m.cursor
	if cursor < 0 || cursor >= len(messages) {
		return
	}

	if ref := messages[cursor].message.ReferencedMessage; ref != nil {
		refIdx := slices.IndexFunc(messages, func(m messageItem) bool {
			return m.separator == nil && m.message.ID == ref.ID
		})
		if refIdx != -1 {
			m.setCursor(refIdx)
		}
	}
}

// SelectLastOwn selects the last message in the selected channel that the user can edit, and returns it.
func (m *Model) SelectLastOwn() (discord.Message, bool) {
	channel := m.channel
	if channel == nil {
		return discord.Message{}, false
	}
	for i, item := range slices.Backward(m.items) {
		message := item.message
		if item.separator != nil || message.ChannelID != channel.ID || !ui.IsMe(m.state, message.Author.ID) ||
			!message.ID.IsValid() || message.Content == "" || len(message.MessageSnapshots) > 0 ||
			(message.Type != discord.DefaultMessage && message.Type != discord.InlinedReplyMessage) {
			continue
		}
		m.setCursor(i)
		return message, true
	}
	return discord.Message{}, false
}

func (m Model) canDeleteMessage(message discord.Message) bool {
	return ui.IsMe(m.state, message.Author.ID) ||
		(message.GuildID.IsValid() && m.state.HasPermissions(message.ChannelID, discord.PermissionManageMessages))
}

func (m Model) InvalidateRendered() {
	for i := range m.items {
		m.items[i].view, m.items[i].root = nil, nil
	}
}

// contentRoot returns the parsed markdown of the content of the item's message, parsing it the first time, or nil if markdown is disabled.
func (m Model) contentRoot(item *messageItem) ast.Node {
	if item.root == nil && m.cfg.Markdown.Enabled {
		item.root = m.parse(item.message)
	}
	return item.root
}
