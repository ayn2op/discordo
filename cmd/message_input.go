package cmd

import (
	"bytes"
	"fmt"
	"io"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"time"
	"unicode"

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
	cfg *config.Config

	edit            bool
	sendMessageData *api.SendMessageData
	cache           *cache.Cache
	mentionsList    *tview.List
	lastSearch      time.Time
}

func newMessageInput(cfg *config.Config) *messageInput {
	mi := &messageInput{
		TextArea:        tview.NewTextArea(),
		cfg:             cfg,
		sendMessageData: &api.SendMessageData{},
		cache:           cache.NewCache(),
		mentionsList:    tview.NewList(),
	}
	mi.Box = ui.ConfigureBox(mi.Box, &cfg.Theme)
	mi.SetInputCapture(mi.onInputCapture)
	mi.
		SetPlaceholder("Select a channel to start chatting").
		SetPlaceholderStyle(tcell.StyleDefault.Dim(true)).
		SetClipboard(
			func(s string) { clipboard.Write(clipboard.FmtText, []byte(s)) },
			func() string { return string(clipboard.Read(clipboard.FmtText)) },
		).
		SetDisabled(true)

	mi.mentionsList.Box = ui.ConfigureBox(mi.mentionsList.Box, &mi.cfg.Theme)
	mi.mentionsList.
		ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)).
		SetTitle("Mentions")

	b := mi.mentionsList.GetBorderSet()
	b.BottomLeft, b.BottomRight = b.BottomT, b.BottomT
	mi.mentionsList.SetBorderSet(b)

	return mi
}

func (mi *messageInput) reset() {
	mi.edit = false
	mi.sendMessageData = &api.SendMessageData{}
	mi.SetTitle("")
	mi.SetText("", true)
}

func (mi *messageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case mi.cfg.Keys.MessageInput.Paste:
		mi.paste()
		return tcell.NewEventKey(tcell.KeyCtrlV, "", tcell.ModNone)

	case mi.cfg.Keys.MessageInput.Send:
		if app.chatView.GetVisibile(mentionsListPageName) {
			mi.tabComplete()
			return nil
		}

		mi.send()
		return nil
	case mi.cfg.Keys.MessageInput.OpenEditor:
		mi.stopTabCompletion()
		mi.editor()
		return nil
	case mi.cfg.Keys.MessageInput.OpenFilePicker:
		mi.stopTabCompletion()
		mi.openFilePicker()
		return nil
	case mi.cfg.Keys.MessageInput.Cancel:
		if app.chatView.GetVisibile(mentionsListPageName) {
			mi.stopTabCompletion()
		} else {
			mi.reset()
		}

		return nil
	case mi.cfg.Keys.MessageInput.TabComplete:
		go app.QueueUpdateDraw(func() { mi.tabComplete() })
		return nil
	}

	if mi.cfg.AutocompleteLimit > 0 {
		if app.chatView.GetVisibile(mentionsListPageName) {
			switch event.Name() {
			case mi.cfg.Keys.MentionsList.Up:
				mi.mentionsList.InputHandler()(tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone), nil)
				return nil
			case mi.cfg.Keys.MentionsList.Down:
				mi.mentionsList.InputHandler()(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone), nil)
				return nil
			}
		}

		go app.QueueUpdateDraw(func() { mi.tabSuggestion() })
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
	if app.chatView.selectedChannel == nil {
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

	text = processText(app.chatView.selectedChannel, []byte(text))

	if mi.edit {
		m, err := app.chatView.messagesList.selectedMessage()
		if err != nil {
			slog.Error("failed to get selected message", "err", err)
			return
		}

		data := api.EditMessageData{Content: option.NewNullableString(text)}
		if _, err := discordState.EditMessageComplex(m.ChannelID, m.ID, data); err != nil {
			slog.Error("failed to edit message", "err", err)
		}

		mi.edit = false
	} else {
		data := mi.sendMessageData
		data.Content = text
		if _, err := discordState.SendMessageComplex(app.chatView.selectedChannel.ID, *data); err != nil {
			slog.Error("failed to send message in channel", "channel_id", app.chatView.selectedChannel.ID, "err", err)
		}
	}

	mi.reset()
	app.chatView.messagesList.Highlight()
	app.chatView.messagesList.ScrollToEnd()
}

func processText(channel *discord.Channel, src []byte) string {
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
		src = slices.Replace(src, rng[0], rng[1], expandMentions(channel, src[rng[0]:rng[1]])...)
	}

	return string(src)
}

func expandMentions(c *discord.Channel, src []byte) []byte {
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
			me, err := discordState.Cabinet.Me()
			if err != nil {
				slog.Error("failed to get client user (me)", "err", err)
			} else if strings.EqualFold(me.Username, name) {
				return []byte(me.ID.Mention())
			}
			return output
		}
		discordState.MemberStore.Each(c.GuildID, func(m *discord.Member) bool {
			if strings.EqualFold(m.User.Username, name) {
				if channelHasUser(c.ID, m.User.ID) {
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

	gID := app.chatView.selectedChannel.GuildID

	if mi.cfg.AutocompleteLimit == 0 {
		if !gID.IsValid() {
			users := app.chatView.selectedChannel.DMRecipients
			res := fuzzy.FindFrom(name, userList(users))
			if len(res) > 0 {
				mi.Replace(pos, posEnd, "@"+users[res[0].Index].Username+" ")
			}
		} else {
			mi.searchMember(gID, name)
			members, err := discordState.Cabinet.Members(gID)
			if err != nil {
				slog.Error("failed to get members from state", "guild_id", gID, "err", err)
				return
			}

			res := fuzzy.FindFrom(name, memberList(members))
			for _, r := range res {
				if channelHasUser(app.chatView.selectedChannel.ID, members[r.Index].User.ID) {
					mi.Replace(pos, posEnd, "@"+members[r.Index].User.Username+" ")
					return
				}
			}
		}
		return
	}
	if mi.mentionsList.GetItemCount() == 0 {
		return
	}
	_, name = mi.mentionsList.GetItemText(mi.mentionsList.GetCurrentItem())
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
	gID := app.chatView.selectedChannel.GuildID
	cID := app.chatView.selectedChannel.ID
	mi.mentionsList.Clear()

	var shown map[string]struct{}
	var userDone struct{}
	if name == "" {
		shown = make(map[string]struct{})
		// Don't show @me in the list of recent authors
		me, err := discordState.Cabinet.Me()
		if err != nil {
			slog.Error("failed to get client user (me)", "err", err)
		} else {
			shown[me.Username] = userDone
		}
	}

	// DMs have recipients, not members
	if !gID.IsValid() {
		if name == "" { // show recent messages' authors
			msgs, err := discordState.Cabinet.Messages(cID)
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
			users := app.chatView.selectedChannel.DMRecipients
			me, err := discordState.Cabinet.Me()
			if err != nil {
				slog.Error("failed to get client user (me)", "err", err)
			} else {
				users = append(users, *me)
			}
			res := fuzzy.FindFrom(name, userList(users))
			for _, r := range res {
				mi.addMentionUser(&users[r.Index])
			}
		}
	} else if name == "" { // show recent messages' authors
		msgs, err := discordState.Cabinet.Messages(cID)
		if err != nil {
			return
		}
		for _, m := range msgs {
			if _, ok := shown[m.Author.Username]; ok {
				continue
			}
			shown[m.Author.Username] = userDone
			discordState.MemberState.RequestMember(gID, m.Author.ID)
			if mem, err := discordState.Cabinet.Member(gID, m.Author.ID); err == nil {
				if mi.addMentionMember(gID, mem) {
					break
				}
			}
		}
	} else {
		mi.searchMember(gID, name)
		mems, err := discordState.Cabinet.Members(gID)
		if err != nil {
			slog.Error("fetching members failed", "err", err)
			return
		}
		res := fuzzy.FindFrom(name, memberList(mems))
		if len(res) > int(mi.cfg.AutocompleteLimit) {
			res = res[:int(mi.cfg.AutocompleteLimit)]
		}
		for _, r := range res {
			if channelHasUser(cID, mems[r.Index].User.ID) &&
				mi.addMentionMember(gID, &mems[r.Index]) {
				break
			}
		}
	}

	if mi.mentionsList.GetItemCount() == 0 {
		mi.stopTabCompletion()
		return
	}

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
func channelHasUser(channelID discord.ChannelID, userID discord.UserID) bool {
	perms, err := discordState.Permissions(channelID, userID)
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
		if c := mi.cache.Get(k); c < discordState.MemberState.SearchLimit {
			mi.cache.Create(key, c)
			return
		}
	}

	// Rate limit on our side because we can't distinguish between a successful search and SearchMember not doing anything because of its internal rate limit that we can't detect
	if mi.lastSearch.Add(discordState.MemberState.SearchFrequency).After(time.Now()) {
		return
	}

	mi.lastSearch = time.Now()
	app.chatView.messagesList.waitForChunkEvent()
	app.chatView.messagesList.setFetchingChunk(true, 0)
	discordState.MemberState.SearchMember(gID, name)
	mi.cache.Create(key, app.chatView.messagesList.waitForChunkEvent())
}

func (mi *messageInput) showMentionList() {
	borders := 0
	if mi.cfg.Theme.Border.Enabled {
		borders = 1
	}
	l := mi.mentionsList
	x, _, _, _ := mi.GetInnerRect()
	_, y, _, _ := mi.GetRect()
	_, _, maxW, maxH := app.chatView.messagesList.GetInnerRect()
	if t := int(mi.cfg.Theme.MentionsList.MaxHeight); t != 0 {
		maxH = min(maxH, t)
	}
	count := l.GetItemCount() + borders
	h := min(count, maxH) + borders + mi.cfg.Theme.Border.Padding[1]
	y -= h
	w := int(mi.cfg.Theme.MentionsList.MinWidth)
	if w == 0 {
		w = maxW
	} else {
		for i := range count - 1 {
			t, _ := mi.mentionsList.GetItemText(i)
			w = max(w, tview.TaggedStringWidth(t))
		}

		w = min(w+borders*2, maxW)
		_, col, _, _ := mi.GetCursor()
		x += min(col, maxW-w)
	}

	l.SetRect(x, y, w, h)

	app.chatView.
		AddAndSwitchToPage(mentionsListPageName, l, false).
		ShowPage(flexPageName)
	app.SetFocus(mi)
}

func (mi *messageInput) addMentionMember(gID discord.GuildID, m *discord.Member) bool {
	if m == nil {
		return false
	}

	name := m.User.DisplayOrUsername()
	if m.Nick != "" {
		name = m.Nick
	}

	// this is WAY faster than discordState.MemberColor
	color, ok := state.MemberColor(m, func(id discord.RoleID) *discord.Role {
		r, _ := discordState.Cabinet.Role(gID, id)
		return r
	})
	if ok {
		name = fmt.Sprintf("[%s]%s[-]", color, name)
	}

	presence, err := discordState.Cabinet.Presence(gID, m.User.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "guild_id", gID, "user_id", m.User.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		name = fmt.Sprintf("[::d]%s[::D]", name)
	}

	mi.mentionsList.AddItem(name, m.User.Username, 0, nil)
	return mi.mentionsList.GetItemCount() > int(mi.cfg.AutocompleteLimit)
}

func (mi *messageInput) addMentionUser(user *discord.User) {
	if user == nil {
		return
	}

	name := user.DisplayOrUsername()
	presence, err := discordState.Cabinet.Presence(discord.NullGuildID, user.ID)
	if err != nil {
		slog.Info("failed to get presence from state", "user_id", user.ID, "err", err)
	} else if presence.Status == discord.OfflineStatus {
		name = fmt.Sprintf("[::d]%s[::D]", name)
	}

	mi.mentionsList.AddItem(name, user.Username, 0, nil)
}

// used by chatView
func (mi *messageInput) removeMentionsList() {
	app.chatView.
		RemovePage(mentionsListPageName).
		SwitchToPage(flexPageName)
}

func (mi *messageInput) stopTabCompletion() {
	if mi.cfg.AutocompleteLimit > 0 {
		mi.mentionsList.Clear()
		mi.removeMentionsList()
		app.SetFocus(mi)
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

	app.Suspend(func() {
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

func (mi *messageInput) addTitle(s string) {
	title := mi.GetTitle()
	if title != "" {
		title += " | "
	}

	mi.SetTitle(title + s)
}

func (mi *messageInput) openFilePicker() {
	if app.chatView.selectedChannel == nil {
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
	mi.addTitle("Attached " + name)
}
