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
	"sort"
	"strings"
	"time"
	"unicode"

	"github.com/ayn2op/discordo/internal/cache"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/sendpart"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/sahilm/fuzzy"
	"github.com/yuin/goldmark/ast"
	"golang.design/x/clipboard"
)

const tmpFilePattern = consts.Name + "_*.md"

var mentionRegex = regexp.MustCompile("@[a-zA-Z0-9._]+")

type messageInput struct {
	*tview.TextArea
	cfg *config.Config

	sendMessageData *api.SendMessageData
	cache           *cache.Cache
	mentionsList    *tview.List
	lastSearch      time.Time
}

type memberList []discord.Member

func newMessageInput(cfg *config.Config) *messageInput {
	mi := &messageInput{
		TextArea: tview.NewTextArea(),
		cfg:      cfg,

		sendMessageData: &api.SendMessageData{},
		cache:           cache.NewCache(),
		mentionsList:    tview.NewList(),
	}

	if err := clipboard.Init(); err != nil {
		slog.Warn("failed to init clipboard", "err", err)
	} else {
		mi.
			SetClipboard(func(s string) {
				clipboard.Write(clipboard.FmtText, []byte(s))
			}, func() string {
				data := clipboard.Read(clipboard.FmtText)
				return string(data)
			})
	}

	mi.Box = ui.ConfigureBox(mi.Box, &cfg.Theme)
	mi.SetInputCapture(mi.onInputCapture)

	mi.mentionsList.Box = ui.ConfigureBox(mi.mentionsList.Box, &mi.cfg.Theme)
	mi.mentionsList.
		ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.Background(tcell.ColorWhite).Foreground(tcell.ColorBlack)).
		SetTitle("Mentions").
		SetRect(0, 0, 0, 0)

	b := mi.mentionsList.GetBorderSet()
	b.BottomLeft, b.BottomRight = b.BottomT, b.BottomT
	mi.mentionsList.SetBorderSet(b)

	return mi
}

func (mi *messageInput) reset() {
	mi.sendMessageData = &api.SendMessageData{}
	mi.SetTitle("")
	mi.SetText("", true)
}

func (mi *messageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case mi.cfg.Keys.MessageInput.Paste:
		mi.paste()
		return tcell.NewEventKey(tcell.KeyCtrlV, 0, tcell.ModNone)

	case mi.cfg.Keys.MessageInput.Send:
		if app.pages.GetVisibile(mentionsListPageName) {
			mi.tabComplete(false)
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
		home, err := os.UserHomeDir()
		if err != nil {
			slog.Error("Failed to read Home Dir", "home", home, "err", err)
		}
		mi.openFilePicker(home)
		return nil
	case mi.cfg.Keys.MessageInput.Cancel:
		if app.pages.GetVisibile(mentionsListPageName) {
			mi.stopTabCompletion()
		} else {
			mi.reset()
		}

		return nil
	case mi.cfg.Keys.MessageInput.TabComplete:
		go app.QueueUpdateDraw(func() { mi.tabComplete(false) })
		return nil
	}

	if mi.cfg.AutocompleteLimit > 0 {
		if app.pages.GetVisibile(mentionsListPageName) {
			count := mi.mentionsList.GetItemCount()
			cur := mi.mentionsList.GetCurrentItem()
			switch event.Name() {
			case mi.cfg.Keys.MentionsList.Down:
				mi.mentionsList.SetCurrentItem((cur + 1) % count)
				return nil
			case mi.cfg.Keys.MentionsList.Up:
				if cur == 0 {
					cur = count
				}
				mi.mentionsList.SetCurrentItem(cur - 1)
				return nil
			}
		}

		go app.QueueUpdateDraw(func() { mi.tabComplete(true) })
	}

	return event
}

func (mi *messageInput) paste() {
	if data := clipboard.Read(clipboard.FmtImage); data != nil {
		name := "clipboard.png"
		mi.attach(name, sendpart.File{Name: name, Reader: bytes.NewReader(data)})
	}
}

func (mi *messageInput) send() {
	if !app.guildsTree.selectedChannelID.IsValid() {
		return
	}

	data := mi.sendMessageData
	if text := strings.TrimSpace(mi.GetText()); text != "" {
		data.Content = processText(app.guildsTree.selectedChannelID, []byte(text))
	}

	if _, err := discordState.SendMessageComplex(app.guildsTree.selectedChannelID, *data); err != nil {
		slog.Error("failed to send message in channel", "channel_id", app.guildsTree.selectedChannelID, "err", err)
	}

	// Close the attached files after sending the message.
	for _, file := range mi.sendMessageData.Files {
		if closer, ok := file.Reader.(io.Closer); ok {
			closer.Close()
		}
	}

	mi.reset()
	app.messagesList.Highlight()
	app.messagesList.ScrollToEnd()
}

func processText(cID discord.ChannelID, src []byte) string {
	var (
		ranges     [][2]int
		canMention = true
	)

	ast.Walk(discordmd.Parse(src), func(node ast.Node, enter bool) (ast.WalkStatus, error) {
		switch node := node.(type) {
		case *ast.CodeBlock:
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
		src = slices.Replace(src, rng[0], rng[1], expandMentions(cID, src[rng[0]:rng[1]])...)
	}

	return string(src)
}

func expandMentions(cID discord.ChannelID, src []byte) []byte {
	return mentionRegex.ReplaceAllFunc(src, func(input []byte) []byte {
		output := input
		name := strings.ToLower(string(input[1:]))
		discordState.MemberStore.Each(app.guildsTree.selectedGuildID, func(m *discord.Member) bool {
			if strings.ToLower(m.User.Username) == name && channelHasUser(cID, m.User.ID) {
				output = []byte(m.User.ID.Mention())
				return true
			}

			return false
		})

		return output
	})
}

func (mi *messageInput) tabComplete(isAuto bool) {
	posEnd, name, r := mi.GetWordUnderCursor(func(r rune) bool {
		return unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_' || r == '.'
	})
	if r != '@' {
		mi.stopTabCompletion()
		return
	}
	pos := posEnd - (len(name) + 1)

	gID := app.guildsTree.selectedGuildID
	cID := app.guildsTree.selectedChannelID

	if !isAuto {
		if mi.cfg.AutocompleteLimit == 0 {
			mi.searchMember(gID, name)

			members, err := discordState.Cabinet.Members(gID)
			if err != nil {
				slog.Error("failed to get members from state", "guild_id", gID, "err", err)
				return
			}

			res := fuzzy.FindFrom(name, memberList(members))
			for _, r := range res {
				if channelHasUser(cID, members[r.Index].User.ID) {
					mi.Replace(pos, posEnd, "@"+members[r.Index].User.Username+" ")
					return
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
		return
	}

	// Special case, show recent messages' authors
	if name == "" {
		msgs, err := discordState.Cabinet.Messages(cID)
		if err != nil {
			return
		}
		shown := make(map[string]bool)
		mi.mentionsList.Clear()
		for _, m := range msgs {
			if shown[m.Author.Username] {
				continue
			}
			shown[m.Author.Username] = true
			discordState.MemberState.RequestMember(gID, m.Author.ID)
			if mem, err := discordState.Cabinet.Member(gID, m.Author.ID); err == nil {
				if mi.addMentionItem(gID, mem) {
					break
				}
			}
		}
	} else {
		mi.searchMember(gID, name)
		mi.mentionsList.Clear()
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
				mi.addMentionItem(gID, &mems[r.Index]) {
				break
			}
		}
	}

	if mi.mentionsList.GetItemCount() == 0 {
		mi.stopTabCompletion()
		return
	}

	_, col, _, _ := mi.GetCursor()
	mi.showMentionList(col - 1)
}

func (m memberList) String(i int) string { return m[i].Nick + m[i].User.DisplayName + m[i].User.Tag() }
func (m memberList) Len() int            { return len(m) }

func channelHasUser(cID discord.ChannelID, id discord.UserID) bool {
	perms, err := discordState.Permissions(cID, id)
	if err != nil {
		slog.Error("can't get permissions", "channel", cID, "user", id)
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
	app.messagesList.waitForChunkEvent()
	app.messagesList.setFetchingChunk(true, 0)
	discordState.MemberState.SearchMember(gID, name)
	mi.cache.Create(key, app.messagesList.waitForChunkEvent())
}

func (mi *messageInput) showMentionList(col int) {
	borders := 0
	if mi.cfg.Theme.Border.Enabled {
		borders = 1
	}
	l := mi.mentionsList
	x, _, _, _ := mi.GetInnerRect()
	_, y, _, _ := mi.GetRect()
	_, _, maxW, maxH := app.messagesList.GetInnerRect()
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
		x += min(col, maxW-w)
	}

	l.SetRect(x, y, w, h)

	app.pages.
		AddAndSwitchToPage(mentionsListPageName, l, false).
		ShowPage(flexPageName)
	app.SetFocus(mi)
}

func (mi *messageInput) addMentionItem(gID discord.GuildID, m *discord.Member) bool {
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
	}

	if presence != nil && presence.Status == discord.OfflineStatus {
		name = fmt.Sprintf("[::d]%s[::D]", name)
	}

	mi.mentionsList.AddItem(name, m.User.Username, 0, nil)
	return mi.mentionsList.GetItemCount() > int(mi.cfg.AutocompleteLimit)
}

func (mi *messageInput) removeMentionsList() {
	app.pages.
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

func (mi *messageInput) openFilePicker(startPath string) {
	if !app.guildsTree.selectedChannelID.IsValid() {
		return
	}

	list := tview.NewList().
		SetWrapAround(true).
		SetHighlightFullLine(true).
		ShowSecondaryText(false).
		SetDoneFunc(func() {
			app.pages.RemovePage(filePickerPageName).SwitchToPage(flexPageName)
			app.SetFocus(mi)
		})

	list.SetTitle("File Picker")

	searchBar := tview.NewInputField().
		SetLabel("/")
	searchBar.SetDoneFunc(func(key tcell.Key) {
		switch key {
		case tcell.KeyEnter:
			app.SetFocus(list)
		case tcell.KeyEscape:
			searchBar.SetText("")
			app.SetFocus(list)
		}
	})

	files, err := os.ReadDir(startPath)
	if err != nil {
		slog.Error("Failed to read Dir", "dir", startPath, "err", err)
	}

	// sort dir first because......why not?
	sort.Slice(files, func(i, j int) bool {
		dirI, dirJ := files[i].IsDir(), files[j].IsDir()
		if dirI != dirJ {
			return dirI
		}
		return files[i].Name() < files[j].Name()
	})

	entries := make([]string, len(files))
	for i, e := range files {
		entries[i] = e.Name()
	}

	updateList := func(filter string) {
		list.Clear()

		parentDir := filepath.Dir(startPath)
		if parentDir != startPath && filter == "" {
			list.AddItem("[::b]..[::-]", "", 0, func() {
				mi.openFilePicker(parentDir)
			})
		}

		var visibleFiles []int
		if filter == "" {
			visibleFiles = make([]int, len(files))
			for i := range files {
				visibleFiles[i] = i
			}
		} else {
			matches := fuzzy.Find(filter, entries)
			visibleFiles = make([]int, len(matches))
			for i, m := range matches {
				visibleFiles[i] = m.Index
			}
		}

		for _, index := range visibleFiles {
			e := files[index]
			fullpath := filepath.Join(startPath, e.Name())
			name := e.Name()

			if e.IsDir() {
				list.AddItem("[::b]"+name+"[::-]", "", 0, func() {
					mi.openFilePicker(fullpath)
				})
			} else {
				list.AddItem(name, "", 0, func() {
					file, err := os.Open(fullpath)
					if err != nil {
						slog.Error("Failed to open file", "file", file, "path", fullpath)
					}
					mi.attach(name, sendpart.File{Name: name, Reader: file})
				})
			}
		}
	}

	searchBar.SetChangedFunc(func(text string) {
		updateList(text)
	})

	updateList("")

	flex := tview.NewFlex().SetDirection(tview.FlexRow).
		AddItem(list, 0, 1, true).
		AddItem(searchBar, 3, 0, false)

	ui.ConfigureBox(list.Box, &mi.cfg.Theme)
	ui.ConfigureBox(searchBar.Box, &mi.cfg.Theme)

	list.SetInputCapture(func(ev *tcell.EventKey) *tcell.EventKey {
		switch ev.Name() {
		case "Rune[/]":
			app.SetFocus(searchBar)
			return nil
		case mi.cfg.Keys.MessagesList.SelectPrevious:
			return tcell.NewEventKey(tcell.KeyUp, 0, tcell.ModNone)
		case mi.cfg.Keys.MessagesList.SelectNext:
			return tcell.NewEventKey(tcell.KeyDown, 0, tcell.ModNone)
		case mi.cfg.Keys.MessagesList.SelectFirst:
			return tcell.NewEventKey(tcell.KeyHome, 0, tcell.ModNone)
		case mi.cfg.Keys.MessagesList.SelectLast:
			return tcell.NewEventKey(tcell.KeyEnd, 0, tcell.ModNone)
		}
		return ev
	})

	app.pages.
		RemovePage(filePickerPageName).
		AddAndSwitchToPage(filePickerPageName, ui.Centered(flex, 40, 25), true).
		ShowPage(flexPageName)
}

func (mi *messageInput) attach(name string, file sendpart.File) {
	mi.sendMessageData.Files = append(mi.sendMessageData.Files, file)
	mi.addTitle("Attached " + name)
}
