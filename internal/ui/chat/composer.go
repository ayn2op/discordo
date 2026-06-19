package chat

import (
	"bytes"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"

	"github.com/ayn2op/discordo/internal/cache"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/help"
	"github.com/ayn2op/tview/keybind"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/diamondburned/ningen/v3"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v3"
	"github.com/ncruces/zenity"
	"github.com/sahilm/fuzzy"
	"github.com/yuin/goldmark/ast"
	"golang.design/x/clipboard"
)

const (
	tmpFilePattern      = consts.Name + "_*.md"
	imageAttachmentName = "clipboard.png"
)

var mentionRegex = regexp.MustCompile("@[a-zA-Z0-9._]+")

type composer struct {
	*tview.TextArea
	chat *Model

	cfg *config.Config

	edit            bool
	sendMessageData *api.SendMessageData
	cache           *cache.Cache
	mentionsList    *mentionsList
	lastSearch      time.Time

	typingTimerMu sync.Mutex
	typingTimer   *time.Timer
}

type tabSuggestMsg struct{}

var _ help.KeyMap = (*composer)(nil)

func newComposer(cfg *config.Config, chat *Model) *composer {
	c := &composer{
		TextArea:        tview.NewTextArea(),
		cfg:             cfg,
		chat:            chat,
		sendMessageData: &api.SendMessageData{},
		cache:           cache.NewCache(),
		mentionsList:    newMentionsList(cfg),
	}
	c.Box = ui.ConfigureBox(c.Box, &cfg.Theme)
	c.
		SetPlaceholder(tview.NewLine(tview.NewSegment("Select a channel to start chatting", tcell.StyleDefault.Dim(true)))).
		SetClipboard(
			func(s string) {
				if err := clipboard.Write(clipboard.FmtText, []byte(s)); err != nil {
					slog.Error("failed to write clipboard text", "err", err)
				}
			},
			func() string {
				return string(clipboard.Read(clipboard.FmtText))
			},
		).
		SetDisabled(true)

	return c
}

// forwardToTextArea passes a key event to the embedded TextArea then resizes for the new content.
// SetChangedFunc isn't used: it fires inside replace's defer, before truncateLines/findCursor, so it would observe stale state.
func (c *composer) forwardToTextArea(ev *tcell.EventKey) tview.Cmd {
	cmd := c.TextArea.Update(ev)
	c.resizeForContent()
	return cmd
}

// resizeForContent fits the input to its newline count, capped at Composer.MaxHeight and at the parent flex's height (leaving at least 1 row for the messages list), then pins the scroll offset to the new visible window.
// Safe to call after every edit; ResizeItem and SetOffset are both idempotent.
func (c *composer) resizeForContent() {
	if c.chat == nil || c.chat.rightFlex == nil {
		return
	}
	_, _, _, outerH := c.Rect()
	_, _, _, innerH := c.InnerRect()
	frame := outerH - innerH // border + title/footer + padding rows
	_, _, _, parentH := c.chat.rightFlex.InnerRect()

	visible := min(
		strings.Count(c.Text(), "\n")+1,  // newline-driven height
		max(c.cfg.Composer.MaxHeight, 1), // user-configured cap
		max(parentH-frame-1, 1),          // available room, reserving 1 row for messages
	)
	c.chat.rightFlex.ResizeItem(c, visible+frame, 1) // outer height = inner content + frame

	// Sync the textarea's inner height so the next keystroke's cursor clamping uses the new size.
	c.SetVisibleSize(0, visible)

	// Clamp scroll: when content overflows, pin the bottom row to the end of the text so backspacing the last newline doesn't leave a blank trailing row; once everything fits, reset to (0,0).
	total := c.LineCount(0)
	row, col := c.Offset()
	maxOff := max(total-visible, 0) // last valid rowOffset that still shows real content on the bottom row
	if row > maxOff {
		c.SetOffset(maxOff, col)
	} else if total <= visible && (row != 0 || col != 0) {
		c.SetOffset(0, 0)
	}
}

func (c *composer) reset() {
	c.edit = false
	c.sendMessageData = &api.SendMessageData{}
	c.SetTitle("")
	c.SetFooter("")
	c.SetText("", true)
}

// The following overrides wrap the embedded TextArea/Box methods to auto-resize when callers change text or chrome.

func (c *composer) SetText(text string, cursorAtTheEnd bool) *tview.TextArea {
	defer c.resizeForContent()
	return c.TextArea.SetText(text, cursorAtTheEnd)
}

func (c *composer) Replace(start, end int, text string) *tview.TextArea {
	defer c.resizeForContent()
	return c.TextArea.Replace(start, end, text)
}

func (c *composer) SetTitle(title string) *tview.Box {
	defer c.resizeForContent()
	return c.Box.SetTitle(title)
}

func (c *composer) SetFooter(footer string) *tview.Box {
	defer c.resizeForContent()
	return c.Box.SetFooter(footer)
}

func (c *composer) stopTypingTimer() {
	c.typingTimerMu.Lock()
	defer c.typingTimerMu.Unlock()
	if c.typingTimer != nil {
		c.typingTimer.Stop()
		c.typingTimer = nil
	}
}

func (c *composer) Update(msg tview.Msg) tview.Cmd {
	switch msg := msg.(type) {
	case tabSuggestMsg:
		return c.tabSuggest()
	case imagePastedMsg:
		if len(msg) == 0 {
			return nil
		}
		c.attach(imageAttachmentName, bytes.NewReader(msg))
		return nil
	case filesPickedMsg:
		selectedChannel, ok := c.chat.SelectedChannel()
		if !ok || selectedChannel.ID != msg.channelID {
			return closeFiles(msg.files)
		}
		for _, file := range msg.files {
			c.attach(file.Name, file.Reader)
		}
		return nil

	case tview.KeyMsg:
		switch {
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Paste.Keybind):
			return tview.Sequence(c.pasteImage(), c.forwardToTextArea(tcell.NewEventKey(tcell.KeyCtrlV, "", tcell.ModNone)))
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Newline.Keybind):
			return c.forwardToTextArea(tcell.NewEventKey(tcell.KeyEnter, "", tcell.ModNone))
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Send.Keybind):
			if c.chat.GetVisible(mentionsListLayerName) {
				return c.tabComplete()
			}
			return c.send()
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.OpenEditor.Keybind):
			cmd := c.stopTabCompletion()
			c.editor()
			return cmd
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.OpenFilePicker.Keybind):
			return tview.Sequence(c.stopTabCompletion(), c.pickFiles())
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Cancel.Keybind):
			if c.chat.GetVisible(mentionsListLayerName) {
				return c.stopTabCompletion()
			}
			c.reset()
			return nil
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.TabComplete.Keybind):
			if c.chat.GetVisible(mentionsListLayerName) {
				return c.tabComplete()
			}
			return c.forwardToTextArea(msg)
		case keybind.Matches(msg, c.cfg.Keybinds.Composer.Undo.Keybind):
			return c.forwardToTextArea(tcell.NewEventKey(tcell.KeyCtrlZ, "", tcell.ModNone))
		}

		typingCmd := c.sendTyping()

		if c.cfg.AutocompleteLimit > 0 {
			if c.chat.GetVisible(mentionsListLayerName) {
				keybinds := c.cfg.Keybinds.MentionsList
				if keybind.Matches(msg, keybinds.SelectUp.Keybind) ||
					keybind.Matches(msg, keybinds.SelectDown.Keybind) ||
					keybind.Matches(msg, keybinds.SelectTop.Keybind) ||
					keybind.Matches(msg, keybinds.SelectBottom.Keybind) {
					return tview.Batch(typingCmd, c.mentionsList.Update(msg))
				}
			}

			// Apply key edits first, then recompute autocomplete through Msg/Cmd.
			return tview.Batch(typingCmd, tview.Sequence(c.forwardToTextArea(msg), c.tabSuggest()))
		}
		return tview.Batch(typingCmd, c.forwardToTextArea(msg))
	}
	return c.TextArea.Update(msg)
}

type imagePastedMsg []byte

func (c *composer) pasteImage() tview.Cmd {
	return func() tview.Msg {
		data := clipboard.Read(clipboard.FmtImage)
		return imagePastedMsg(data)
	}
}

type filesPickedMsg struct {
	channelID discord.ChannelID
	files     []sendpart.File
}

func (c *composer) pickFiles() tview.Cmd {
	selectedChannel, ok := c.chat.SelectedChannel()
	if !ok {
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

func (c *composer) sendTyping() tview.Cmd {
	if !c.cfg.TypingIndicator.Send {
		return nil
	}

	c.typingTimerMu.Lock()
	if c.typingTimer != nil {
		c.typingTimerMu.Unlock()
		return nil
	}
	c.typingTimer = time.AfterFunc(typingDuration, func() {
		c.typingTimerMu.Lock()
		c.typingTimer = nil
		c.typingTimerMu.Unlock()
	})
	c.typingTimerMu.Unlock()

	selectedChannel, ok := c.chat.SelectedChannel()
	if !ok {
		return nil
	}
	channelID := selectedChannel.ID
	return func() tview.Msg {
		c.chat.state.Typing(channelID)
		return nil
	}
}

func (c *composer) send() tview.Cmd {
	selectedChannel, ok := c.chat.SelectedChannel()
	if !ok {
		return nil
	}

	text := strings.TrimSpace(c.Text())
	if text == "" && len(c.sendMessageData.Files) == 0 {
		return nil
	}

	text = c.processText(selectedChannel, []byte(text))
	data := *c.sendMessageData
	data.Files = slices.Clone(data.Files)

	var editMessage discord.Message
	edit := c.edit
	if edit {
		selectedMessage, ok := c.chat.messagesList.selectedMessage()
		if !ok {
			return nil
		}
		editMessage = *selectedMessage
	}

	c.stopTypingTimer()
	c.reset()
	c.chat.messagesList.clearSelection()
	c.chat.messagesList.ScrollBottom()

	return func() tview.Msg {
		defer closeFiles(data.Files)()
		if edit {
			editData := api.EditMessageData{Content: option.NewNullableString(text)}
			if _, err := c.chat.state.EditMessageComplex(editMessage.ChannelID, editMessage.ID, editData); err != nil {
				slog.Error("failed to edit message", "err", err)
			}
			return nil
		}
		data.Content = text
		if _, err := c.chat.state.SendMessageComplex(selectedChannel.ID, data); err != nil {
			slog.Error("failed to send message in channel", "channel_id", selectedChannel.ID, "err", err)
		}
		return nil
	}
}

func (c *composer) processText(channel *discord.Channel, src []byte) string {
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

	ast.Walk(discordmd.Parse(src), func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		switch node := node.(type) {
		case *ast.CodeBlock, *ast.FencedCodeBlock:
			canMention = !enter
		case *discordmd.Inline:
			if (node.Attr & discordmd.AttrMonospace) != 0 {
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

func (c *composer) expandMentions(channel *discord.Channel, src []byte) []byte {
	state := c.chat.state
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

func isMentionChar(r rune) bool {
	return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
}

func (c *composer) tabComplete() tview.Cmd {
	posEnd, name, r := c.GetWordUnderCursor(isMentionChar)
	if r != '@' {
		return c.stopTabCompletion()
	}
	pos := posEnd - (len(name) + 1)

	selectedChannel, ok := c.chat.SelectedChannel()
	if !ok {
		return nil
	}
	gID := selectedChannel.GuildID

	if c.cfg.AutocompleteLimit == 0 {
		if !gID.IsValid() {
			users := selectedChannel.DMRecipients
			res := fuzzy.FindFrom(name, userList(users))
			if len(res) > 0 {
				c.Replace(pos, posEnd, "@"+users[res[0].Index].Username+" ")
			}
		} else {
			cmd := c.searchMember(gID, name)
			members, err := c.chat.state.Cabinet.Members(gID)
			if err != nil {
				slog.Error("failed to get members from state", "guild_id", gID, "err", err)
				return cmd
			}

			res := fuzzy.FindFrom(name, memberList(members))
			for _, r := range res {
				if channelHasUser(c.chat.state, selectedChannel.ID, members[r.Index].User.ID) {
					c.Replace(pos, posEnd, "@"+members[r.Index].User.Username+" ")
					return cmd
				}
			}
			return cmd
		}
		return nil
	}
	if c.mentionsList.itemCount() == 0 {
		return nil
	}
	name, ok = c.mentionsList.selectedInsertText()
	if !ok {
		return nil
	}
	c.Replace(pos, posEnd, "@"+name+" ")
	return c.stopTabCompletion()
}

func (c *composer) tabSuggest() tview.Cmd {
	_, name, r := c.GetWordUnderCursor(isMentionChar)
	if r != '@' {
		return c.stopTabCompletion()
	}
	selectedChannel, ok := c.chat.SelectedChannel()
	if !ok {
		return nil
	}
	gID := selectedChannel.GuildID
	cID := selectedChannel.ID
	c.mentionsList.clear()

	var shown map[string]struct{}
	var userDone struct{}
	if name == "" {
		shown = make(map[string]struct{})
		// Don't show @me in the list of recent authors
		me, _ := c.chat.state.Cabinet.Me()
		shown[me.Username] = userDone
	}

	// DMs have recipients, not members
	if !gID.IsValid() {
		if name == "" { // show recent messages' authors
			msgs, err := c.chat.state.Cabinet.Messages(cID)
			if err != nil {
				return nil
			}
			for _, m := range msgs {
				if _, ok := shown[m.Author.Username]; ok {
					continue
				}
				shown[m.Author.Username] = userDone
				c.addMentionUser(&m.Author)
			}
		} else {
			users := selectedChannel.DMRecipients
			me, _ := c.chat.state.Cabinet.Me()
			users = append(users, *me)
			res := fuzzy.FindFrom(name, userList(users))
			for _, r := range res {
				c.addMentionUser(&users[r.Index])
			}
		}
	} else if name == "" { // show recent messages' authors
		msgs, err := c.chat.state.Cabinet.Messages(cID)
		if err != nil {
			return nil
		}
		for _, m := range msgs {
			if _, ok := shown[m.Author.Username]; ok {
				continue
			}
			shown[m.Author.Username] = userDone
			c.chat.state.MemberState.RequestMember(gID, m.Author.ID)
			if mem, err := c.chat.state.Cabinet.Member(gID, m.Author.ID); err == nil {
				if c.addMentionMember(gID, mem) {
					break
				}
			}
		}
	} else {
		searchCmd := c.searchMember(gID, name)
		mems, err := c.chat.state.Cabinet.Members(gID)
		if err != nil {
			slog.Error("fetching members failed", "err", err)
			return searchCmd
		}
		res := fuzzy.FindFrom(name, memberList(mems))
		if len(res) > int(c.cfg.AutocompleteLimit) {
			res = res[:int(c.cfg.AutocompleteLimit)]
		}
		for _, r := range res {
			if channelHasUser(c.chat.state, cID, mems[r.Index].User.ID) &&
				c.addMentionMember(gID, &mems[r.Index]) {
				break
			}
		}
		if c.mentionsList.itemCount() == 0 {
			return tview.Batch(c.stopTabCompletion(), searchCmd)
		}
	}

	if c.mentionsList.itemCount() == 0 {
		return c.stopTabCompletion()
	}

	c.mentionsList.rebuild()
	return c.showMentionsList()
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
func (c *composer) searchMember(gID discord.GuildID, name string) tview.Cmd {
	key := gID.String() + " " + name
	if c.cache.Exists(key) {
		return nil
	}
	// If searching for "ab" returns less than SearchLimit,
	// then "abc" would not return anything new because we already searched
	// everything starting with "ab". This will still be true even if a new
	// member joins because arikawa loads new members into the state.
	if k := key[:len(key)-1]; c.cache.Exists(k) {
		if count := c.cache.Get(k); count < c.chat.state.MemberState.SearchLimit {
			c.cache.Create(key, count)
			return nil
		}
	}

	// Rate limit on our side because we can't distinguish between a successful search and SearchMember not doing anything because of its internal rate limit that we can't detect
	if c.lastSearch.Add(c.chat.state.MemberState.SearchFrequency).After(time.Now()) {
		return nil
	}

	c.lastSearch = time.Now()
	return func() tview.Msg {
		c.chat.messagesList.waitForChunkEvent()
		c.chat.messagesList.setFetchingChunk(true, 0)
		c.chat.state.MemberState.SearchMember(gID, name)
		c.cache.Create(key, c.chat.messagesList.waitForChunkEvent())
		return tabSuggestMsg{}
	}
}

func (c *composer) showMentionsList() tview.Cmd {
	borders := 0
	if c.cfg.Theme.Border.Enabled {
		borders = 1
	}
	l := c.mentionsList
	x, _, _, _ := c.InnerRect()
	_, y, _, _ := c.Rect()
	_, _, maxW, maxH := c.chat.messagesList.InnerRect()
	if t := int(c.cfg.Theme.MentionsList.MaxHeight); t != 0 {
		maxH = min(maxH, t)
	}
	count := c.mentionsList.itemCount() + borders
	h := min(count, maxH) + borders + c.cfg.Theme.Border.Padding[1]
	y -= h
	w := int(c.cfg.Theme.MentionsList.MinWidth)
	if w == 0 {
		w = maxW
	} else {
		w = max(w, c.mentionsList.maxDisplayWidth())

		w = min(w+borders*2, maxW)
		_, col, _, _ := c.GetCursor()
		x += min(col, maxW-w)
	}

	l.SetRect(x, y, w, h)
	c.chat.ShowLayer(mentionsListLayerName).SendToFront(mentionsListLayerName)
	return tview.SetFocus(c)
}

func (c *composer) addMentionMember(gID discord.GuildID, m *discord.Member) bool {
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
		r, _ := c.chat.state.Cabinet.Role(gID, id)
		return r
	})
	if ok {
		style = style.Foreground(tcell.NewHexColor(int32(color)))
	}

	presence, err := c.chat.state.Cabinet.Presence(gID, m.User.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "guild_id", gID, "user_id", m.User.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		style = style.Dim(true)
	}

	c.mentionsList.append(mentionsListItem{
		insertText:  m.User.Username,
		displayText: name,
		style:       style,
	})
	return c.mentionsList.itemCount() > int(c.cfg.AutocompleteLimit)
}

func (c *composer) addMentionUser(user *discord.User) {
	if user == nil {
		return
	}

	name := user.DisplayOrUsername()
	style := tcell.StyleDefault
	presence, err := c.chat.state.Cabinet.Presence(discord.NullGuildID, user.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "user_id", user.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		style = style.Dim(true)
	}

	c.mentionsList.append(mentionsListItem{
		insertText:  user.Username,
		displayText: name,
		style:       style,
	})
}

func (c *composer) removeMentionsList() {
	// Make sure that the layer is visible before hiding it to avoid a refocus in the parent.
	if c.chat.GetVisible(mentionsListLayerName) {
		c.chat.HideLayer(mentionsListLayerName)
	}
}

func (c *composer) stopTabCompletion() tview.Cmd {
	if c.cfg.AutocompleteLimit > 0 {
		c.mentionsList.clear()
		c.removeMentionsList()
		return tview.SetFocus(c)
	}
	return nil
}

func (c *composer) editor() {
	file, err := os.CreateTemp("", tmpFilePattern)
	if err != nil {
		slog.Error("failed to create tmp file", "err", err)
		return
	}
	defer file.Close()
	defer os.Remove(file.Name())

	file.WriteString(c.Text())

	if c.cfg.Editor == "" {
		slog.Warn("Attempt to open file with editor, but no editor is set")
		return
	}

	cmd := c.cfg.CreateEditorCommand(file.Name())
	if cmd == nil {
		return
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	c.chat.app.Suspend(func() {
		err := cmd.Run()
		if err != nil {
			slog.Error("failed to run command", "args", cmd.Args, "err", err)
			return
		}
	})

	msg, err := os.ReadFile(file.Name())
	if err != nil {
		slog.Error("failed to read tmp file", "name", file.Name(), "err", err)
		return
	}

	c.SetText(strings.TrimSpace(string(msg)), true)
}

func (c *composer) attach(name string, reader io.Reader) {
	c.sendMessageData.Files = append(c.sendMessageData.Files, sendpart.File{Name: name, Reader: reader})

	var names []string
	for _, file := range c.sendMessageData.Files {
		names = append(names, file.Name)
	}
	c.SetFooter("Attached " + humanJoin(names))
}

func (c *composer) canAttachFiles() bool {
	selectedChannel, ok := c.chat.SelectedChannel()
	return ok && c.chat.state.HasPermissions(selectedChannel.ID, discord.PermissionAttachFiles)
}

func (c *composer) ShortHelp() []keybind.Keybind {
	if c.chat.GetVisible(mentionsListLayerName) {
		cfg := c.cfg.Keybinds.MentionsList
		ccfg := c.cfg.Keybinds.Composer
		short := []keybind.Keybind{cfg.SelectUp.Keybind, cfg.SelectDown.Keybind, ccfg.Cancel.Keybind}
		if c.canAttachFiles() {
			short = append(short, ccfg.OpenFilePicker.Keybind)
		}
		return short
	}

	cfg := c.cfg.Keybinds.Composer
	short := []keybind.Keybind{cfg.Send.Keybind, cfg.Newline.Keybind, cfg.Cancel.Keybind, cfg.Paste.Keybind, cfg.OpenEditor.Keybind}
	if c.canAttachFiles() {
		short = append(short, cfg.OpenFilePicker.Keybind)
	}
	return short
}

func (c *composer) FullHelp() [][]keybind.Keybind {
	if c.chat.GetVisible(mentionsListLayerName) {
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

	return [][]keybind.Keybind{
		{cfg.Send.Keybind, cfg.Newline.Keybind, cfg.Cancel.Keybind, cfg.TabComplete.Keybind, cfg.Undo.Keybind},
		openEditor,
	}
}
