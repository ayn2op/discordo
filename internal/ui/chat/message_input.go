package chat

import (
	"bytes"
	"github.com/ayn2op/tview/layers"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"
	"unicode"
	"reflect"

	"github.com/ayn2op/discordo/internal/cache"
	"github.com/ayn2op/discordo/internal/clipboard"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
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
)

const tmpFilePattern = consts.Name + "_*.md"

var mentionRegex = regexp.MustCompile("@[a-zA-Z0-9._]+")

type messageInput struct {
	*tview.TextArea
	cfg      *config.Config
	chatView *View

	edit            bool
	sendMessageData *api.SendMessageData
	cache           *cache.Cache
	mentionsList    *mentionsList
	lastSearch      time.Time

	typingTimerMu sync.Mutex
	typingTimer   *time.Timer

	hotkeysShowMap map[string]func() bool
}

func newMessageInput(cfg *config.Config, chatView *View) *messageInput {
	mi := &messageInput{
		TextArea:        tview.NewTextArea(),
		cfg:             cfg,
		chatView:        chatView,
		sendMessageData: &api.SendMessageData{},
		cache:           cache.NewCache(),
	}
	mi.mentionsList = newMentionsList(mi)
	mi.Box = ui.ConfigureBox(mi.Box, &cfg.Theme)
	mi.SetInputCapture(mi.onInputCapture)
	mi.
		SetPlaceholder(tview.NewLine(tview.NewSegment("Select a channel to start chatting", tcell.StyleDefault.Dim(true)))).
		SetClipboard(
			func(s string) { clipboard.Write(clipboard.FmtText, []byte(s)) },
			func() string { return string(clipboard.Read(clipboard.FmtText)) },
		).
		SetDisabled(true)

	mi.hotkeysShowMap = map[string]func() bool{
		"attach": mi.hkAttach,
	}

	return mi
}

func (mi *messageInput) reset() {
	mi.edit = false
	mi.sendMessageData = &api.SendMessageData{}
	mi.SetTitle("")
	mi.SetFooter("")
	mi.SetText("", true)
}

func (mi *messageInput) stopTypingTimer() {
	if mi.typingTimer != nil {
		mi.typingTimer.Stop()
		mi.typingTimer = nil
	}
}

func (mi *messageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case mi.cfg.Keybinds.MessageInput.Paste:
		mi.paste()
		return tcell.NewEventKey(tcell.KeyCtrlV, "", tcell.ModNone)

	case mi.cfg.Keybinds.MessageInput.Send:
		if mi.chatView.layers.GetVisible(mentionsListLayerName) {
			mi.tabComplete()
			return nil
		}

		mi.send()
		return nil
	case mi.cfg.Keybinds.MessageInput.OpenEditor:
		mi.stopTabCompletion()
		mi.editor()
		return nil
	case mi.cfg.Keybinds.MessageInput.OpenFilePicker:
		mi.stopTabCompletion()
		mi.openFilePicker()
		return nil
	case mi.cfg.Keybinds.MessageInput.Cancel:
		if mi.chatView.layers.GetVisible(mentionsListLayerName) {
			mi.stopTabCompletion()
		} else {
			if mi.edit != false ||
			   mi.GetTitle() != "" ||
			   mi.GetFooter() != "" ||
			   mi.GetText() != "" {
				mi.reset()
			} else {
				mi.chatView.app.SetFocus(mi.chatView.hotkeysBar)
			}
		}

		return nil
	case mi.cfg.Keybinds.MessageInput.TabComplete:
		go mi.chatView.app.QueueUpdateDraw(func() { mi.tabComplete() })
		return nil
	default:
		if mi.cfg.TypingIndicator.Send && mi.typingTimer == nil {
			mi.typingTimer = time.AfterFunc(typingDuration, func() {
				mi.typingTimerMu.Lock()
				mi.typingTimer = nil
				mi.typingTimerMu.Unlock()
			})

			if selectedChannel := mi.chatView.SelectedChannel(); selectedChannel != nil {
				go mi.chatView.state.Typing(selectedChannel.ID)
			}
		}
	}

	if mi.cfg.AutocompleteLimit > 0 {
		if mi.chatView.layers.GetVisible(mentionsListLayerName) {
			handler := mi.mentionsList.InputHandler()
			switch event.Name() {
			case mi.cfg.Keybinds.MentionsList.Up:
				handler(tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone), nil)
				return nil
			case mi.cfg.Keybinds.MentionsList.Down:
				handler(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone), nil)
				return nil
			case mi.cfg.Keybinds.MentionsList.Top:
				handler(tcell.NewEventKey(tcell.KeyHome, "", tcell.ModNone), nil)
				return nil
			case mi.cfg.Keybinds.MentionsList.Bottom:
				handler(tcell.NewEventKey(tcell.KeyEnd, "", tcell.ModNone), nil)
				return nil
			}
		}

		go mi.chatView.app.QueueUpdateDraw(func() { mi.tabSuggestion() })
	}

	return event
}

func (mi *messageInput) paste() {
	if data := clipboard.Read(clipboard.FmtImage); data != nil {
		name := "clipboard.png"
		mi.attach(name, bytes.NewReader(data))
	}
}

func (mi *messageInput) send() {
	selected := mi.chatView.SelectedChannel()
	if selected == nil {
		return
	}

	text := strings.TrimSpace(mi.GetText())
	if text == "" && len(mi.sendMessageData.Files) == 0 {
		return
	}

	// Close attached files on return
	defer func() {
		for _, file := range mi.sendMessageData.Files {
			if closer, ok := file.Reader.(io.Closer); ok {
				closer.Close()
			}
		}
	}()

	text = mi.processText(selected, []byte(text))

	if mi.edit {
		m, err := mi.chatView.messagesList.selectedMessage()
		if err != nil {
			slog.Error("failed to get selected message", "err", err)
			return
		}

		data := api.EditMessageData{Content: option.NewNullableString(text)}
		if _, err := mi.chatView.state.EditMessageComplex(m.ChannelID, m.ID, data); err != nil {
			slog.Error("failed to edit message", "err", err)
		}

		mi.edit = false
	} else {
		data := mi.sendMessageData
		data.Content = text
		if _, err := mi.chatView.state.SendMessageComplex(selected.ID, *data); err != nil {
			slog.Error("failed to send message in channel", "channel_id", selected.ID, "err", err)
		}
	}

	if mi.typingTimer != nil {
		mi.typingTimer.Stop()
		mi.typingTimer = nil
	}
	mi.reset()
	mi.chatView.messagesList.clearSelection()
	mi.chatView.messagesList.ScrollToEnd()
}

func (mi *messageInput) processText(channel *discord.Channel, src []byte) string {
	// Fast path: no mentions to expand.
	if bytes.IndexByte(src, '@') == -1 {
		return string(src)
	}

	// Fast path: no back ticks (code blocks), so expand mentions directly.
	if bytes.IndexByte(src, '`') == -1 {
		return string(mi.expandMentions(channel, src))
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
		src = slices.Replace(src, rng[0], rng[1], mi.expandMentions(channel, src[rng[0]:rng[1]])...)
	}

	return string(src)
}

func (mi *messageInput) expandMentions(c *discord.Channel, src []byte) []byte {
	state := mi.chatView.state
	return mentionRegex.ReplaceAllFunc(src, func(input []byte) []byte {
		output := input
		name := string(input[1:])
		if c.Type == discord.DirectMessage || c.Type == discord.GroupDM {
			for _, user := range c.DMRecipients {
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
		state.MemberStore.Each(c.GuildID, func(m *discord.Member) bool {
			if strings.EqualFold(m.User.Username, name) {
				if channelHasUser(state, c.ID, m.User.ID) {
					output = []byte(m.User.ID.Mention())
				}
				return true
			}
			return false
		})
		return output
	})
}

func (mi *messageInput) tabComplete() {
	posEnd, name, r := mi.GetWordUnderCursor(func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
	})
	if r != '@' {
		mi.stopTabCompletion()
		return
	}
	pos := posEnd - (len(name) + 1)

	selected := mi.chatView.SelectedChannel()
	if selected == nil {
		return
	}
	gID := selected.GuildID

	if mi.cfg.AutocompleteLimit == 0 {
		if !gID.IsValid() {
			users := selected.DMRecipients
			res := fuzzy.FindFrom(name, userList(users))
			if len(res) > 0 {
				mi.Replace(pos, posEnd, "@"+users[res[0].Index].Username+" ")
			}
		} else {
			mi.searchMember(gID, name)
			members, err := mi.chatView.state.Cabinet.Members(gID)
			if err != nil {
				slog.Error("failed to get members from state", "guild_id", gID, "err", err)
				return
			}

			res := fuzzy.FindFrom(name, memberList(members))
			for _, r := range res {
				if channelHasUser(mi.chatView.state, selected.ID, members[r.Index].User.ID) {
					mi.Replace(pos, posEnd, "@"+members[r.Index].User.Username+" ")
					return
				}
			}
		}
		return
	}
	if mi.mentionsList.itemCount() == 0 {
		return
	}
	name, ok := mi.mentionsList.selectedInsertText()
	if !ok {
		return
	}
	mi.Replace(pos, posEnd, "@"+name+" ")
	mi.stopTabCompletion()
}

func (mi *messageInput) tabSuggestion() {
	_, name, r := mi.GetWordUnderCursor(func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
	})
	if r != '@' {
		mi.stopTabCompletion()
		return
	}
	selected := mi.chatView.SelectedChannel()
	if selected == nil {
		return
	}
	gID := selected.GuildID
	cID := selected.ID
	mi.mentionsList.clear()

	var shown map[string]struct{}
	var userDone struct{}
	if name == "" {
		shown = make(map[string]struct{})
		// Don't show @me in the list of recent authors
		me, _ := mi.chatView.state.Cabinet.Me()
		shown[me.Username] = userDone
	}

	// DMs have recipients, not members
	if !gID.IsValid() {
		if name == "" { // show recent messages' authors
			msgs, err := mi.chatView.state.Cabinet.Messages(cID)
			if err != nil {
				return
			}
			for _, m := range msgs {
				if _, ok := shown[m.Author.Username]; ok {
					continue
				}
				shown[m.Author.Username] = userDone
				mi.addMentionUser(&m.Author)
			}
		} else {
			users := selected.DMRecipients
			me, _ := mi.chatView.state.Cabinet.Me()
			users = append(users, *me)
			res := fuzzy.FindFrom(name, userList(users))
			for _, r := range res {
				mi.addMentionUser(&users[r.Index])
			}
		}
	} else if name == "" { // show recent messages' authors
		msgs, err := mi.chatView.state.Cabinet.Messages(cID)
		if err != nil {
			return
		}
		for _, m := range msgs {
			if _, ok := shown[m.Author.Username]; ok {
				continue
			}
			shown[m.Author.Username] = userDone
			mi.chatView.state.MemberState.RequestMember(gID, m.Author.ID)
			if mem, err := mi.chatView.state.Cabinet.Member(gID, m.Author.ID); err == nil {
				if mi.addMentionMember(gID, mem) {
					break
				}
			}
		}
	} else {
		mi.searchMember(gID, name)
		mems, err := mi.chatView.state.Cabinet.Members(gID)
		if err != nil {
			slog.Error("fetching members failed", "err", err)
			return
		}
		res := fuzzy.FindFrom(name, memberList(mems))
		if len(res) > int(mi.cfg.AutocompleteLimit) {
			res = res[:int(mi.cfg.AutocompleteLimit)]
		}
		for _, r := range res {
			if channelHasUser(mi.chatView.state, cID, mems[r.Index].User.ID) &&
				mi.addMentionMember(gID, &mems[r.Index]) {
				break
			}
		}
	}

	if mi.mentionsList.itemCount() == 0 {
		mi.stopTabCompletion()
		return
	}

	mi.mentionsList.rebuild()
	mi.showMentionList()
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

// channelHasUser checks if a user has permission to view the specified channel
func channelHasUser(state *ningen.State, channelID discord.ChannelID, userID discord.UserID) bool {
	perms, err := state.Permissions(channelID, userID)
	if err != nil {
		slog.Error("failed to get permissions", "err", err, "channel", channelID, "user", userID)
		return false
	}
	return perms.Has(discord.PermissionViewChannel)
}

func (mi *messageInput) searchMember(gID discord.GuildID, name string) {
	key := gID.String() + " " + name
	if mi.cache.Exists(key) {
		return
	}
	// If searching for "ab" returns less than SearchLimit,
	// then "abc" would not return anything new because we already searched
	// everything starting with "ab". This will still be true even if a new
	// member joins because arikawa loads new members into the state.
	if k := key[:len(key)-1]; mi.cache.Exists(k) {
		if c := mi.cache.Get(k); c < mi.chatView.state.MemberState.SearchLimit {
			mi.cache.Create(key, c)
			return
		}
	}

	// Rate limit on our side because we can't distinguish between a successful search and SearchMember not doing anything because of its internal rate limit that we can't detect
	if mi.lastSearch.Add(mi.chatView.state.MemberState.SearchFrequency).After(time.Now()) {
		return
	}

	mi.lastSearch = time.Now()
	mi.chatView.messagesList.waitForChunkEvent()
	mi.chatView.messagesList.setFetchingChunk(true, 0)
	mi.chatView.state.MemberState.SearchMember(gID, name)
	mi.cache.Create(key, mi.chatView.messagesList.waitForChunkEvent())
}

func (mi *messageInput) showMentionList() {
	borders := 0
	if mi.cfg.Theme.Border.Enabled {
		borders = 1
	}
	l := mi.mentionsList
	x, _, _, _ := mi.GetInnerRect()
	_, y, _, _ := mi.GetRect()
	_, _, maxW, maxH := mi.chatView.messagesList.GetInnerRect()
	if t := int(mi.cfg.Theme.MentionsList.MaxHeight); t != 0 {
		maxH = min(maxH, t)
	}
	count := mi.mentionsList.itemCount() + borders
	h := min(count, maxH) + borders + mi.cfg.Theme.Border.Padding[1]
	y -= h
	w := int(mi.cfg.Theme.MentionsList.MinWidth)
	if w == 0 {
		w = maxW
	} else {
		w = max(w, mi.mentionsList.maxDisplayWidth())

		w = min(w+borders*2, maxW)
		_, col, _, _ := mi.GetCursor()
		x += min(col-1, maxW-w)
	}

	l.SetRect(x, y, w, h)

	mi.chatView.layers.
		AddLayer(l,
			layers.WithName(mentionsListLayerName),
			layers.WithResize(false),
			layers.WithVisible(true),
		).
		SendToFront(mentionsListLayerName)
	mi.chatView.app.SetFocus(mi)
}

func (mi *messageInput) addMentionMember(gID discord.GuildID, m *discord.Member) bool {
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
		r, _ := mi.chatView.state.Cabinet.Role(gID, id)
		return r
	})
	if ok {
		style = style.Foreground(tcell.NewHexColor(int32(color)))
	}

	presence, err := mi.chatView.state.Cabinet.Presence(gID, m.User.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "guild_id", gID, "user_id", m.User.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		style = style.Dim(true)
	}

	mi.mentionsList.append(mentionsListItem{
		insertText:  name,
		displayText: name,
		style:       style,
	})
	return mi.mentionsList.itemCount() > int(mi.cfg.AutocompleteLimit)
}

func (mi *messageInput) addMentionUser(user *discord.User) {
	if user == nil {
		return
	}

	name := user.DisplayOrUsername()
	style := tcell.StyleDefault
	presence, err := mi.chatView.state.Cabinet.Presence(discord.NullGuildID, user.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "user_id", user.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		style = style.Dim(true)
	}

	mi.mentionsList.append(mentionsListItem{
		insertText:  name,
		displayText: name,
		style:       style,
	})
}

// used by chatView
func (mi *messageInput) removeMentionsList() {
	mi.chatView.layers.RemoveLayer(mentionsListLayerName)
}

func (mi *messageInput) stopTabCompletion() {
	if mi.cfg.AutocompleteLimit > 0 {
		mi.mentionsList.clear()
		mi.removeMentionsList()
		mi.chatView.app.SetFocus(mi)
	}
}

func (mi *messageInput) editor() {
	file, err := os.CreateTemp("", tmpFilePattern)
	if err != nil {
		slog.Error("failed to create tmp file", "err", err)
		return
	}
	defer file.Close()
	defer os.Remove(file.Name())

	file.WriteString(mi.GetText())

	cmd := exec.Command(mi.cfg.Editor, file.Name())
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	mi.chatView.app.Suspend(func() {
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

	mi.SetText(strings.TrimSpace(string(msg)), true)
}

func (mi *messageInput) openFilePicker() {
	if mi.chatView.SelectedChannel() == nil {
		return
	}

	paths, err := zenity.SelectFileMultiple()
	if err != nil {
		slog.Error("failed to open file dialog", "err", err)
		return
	}

	for _, path := range paths {
		file, err := os.Open(path)
		if err != nil {
			slog.Error("failed to open file", "path", path, "err", err)
			continue
		}

		name := filepath.Base(path)
		mi.attach(name, file)
	}
}

func (mi *messageInput) attach(name string, reader io.Reader) {
	mi.sendMessageData.Files = append(mi.sendMessageData.Files, sendpart.File{Name: name, Reader: reader})

	var names []string
	for _, file := range mi.sendMessageData.Files {
		names = append(names, file.Name)
	}
	mi.SetFooter("Attached " + humanJoin(names))
}

// Set hotkeys on focus.
func (mi *messageInput) Focus(delegate func(p tview.Primitive)) {
	mi.hotkeys()
	mi.TextArea.Focus(delegate)
}

// Set hotkeys on mouse focus.
func (mi *messageInput) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return mi.TextArea.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		return mi.TextArea.MouseHandler()(action, event, func(p tview.Primitive) {
			if p == mi.TextArea {
				mi.hotkeys()
			}
			setFocus(p)
		})
	})
}

func (mi *messageInput) hotkeys() {
	if mi.chatView.layers.HasLayer(mentionsListLayerName) {
		mi.mentionsList.hotkeys()
		return
	}
	mi.chatView.hotkeysBar.hotkeysFromValue(
		reflect.ValueOf(mi.cfg.Keybinds.MessageInput),
		mi.hotkeysShowMap,
	)
}

func (ml mentionsList) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return ml.List.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		return ml.List.MouseHandler()(action, event, func(p tview.Primitive) {
			if p == ml.List {
				ml.hotkeys()
			}
			setFocus(p)
		})
	})
}

func (ml *mentionsList) hotkeys() {
	ml.messageInput.chatView.hotkeysBar.hotkeysFromValue(
		reflect.ValueOf(ml.messageInput.cfg.Keybinds.MentionsList),
		nil,
	)
	ml.messageInput.chatView.hotkeysBar.appendHotkeys([]hotkey{
		{name: "select", bind: ml.messageInput.cfg.Keybinds.MessageInput.TabComplete, hot: true},
	})
}

func (mi *messageInput) hkAttach() bool {
	sel := mi.chatView.SelectedChannel()
	return sel != nil && mi.chatView.state.HasPermissions(sel.ID, discord.PermissionAttachFiles)
}
