package cmd

import (
	"log/slog"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/ayn2op/discordo/internal/cache"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/consts"
	"github.com/ayn2op/discordo/internal/ui"
	"github.com/ayn2op/tview"
	"github.com/diamondburned/arikawa/v3/api"
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/diamondburned/arikawa/v3/state"
	"github.com/diamondburned/arikawa/v3/utils/json/option"
	"github.com/diamondburned/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v2"
	"github.com/sahilm/fuzzy"
	"github.com/yuin/goldmark/ast"
)

const tmpFilePattern = consts.Name + "_*.md"

var mentionRegex = regexp.MustCompile("@[a-zA-Z0-9._]+")

type messageInput struct {
	*tview.TextArea
	cfg            *config.Config
	cache          *cache.Cache
	autocomplete   *tview.List
	replyMessageID discord.MessageID
	lastSearch     time.Time
}

type memberList struct {
	cfg *config.Config
	members []discord.Member
}

func newMessageInput(cfg *config.Config) *messageInput {
	mi := &messageInput{
		TextArea:     tview.NewTextArea(),
		cfg:          cfg,
		cache:        cache.NewCache(),
		autocomplete: tview.NewList(),
	}

	mi.Box = ui.NewConfiguredBox(mi.Box, &cfg.Theme)

	mi.
		SetTextStyle(tcell.StyleDefault.Background(tcell.GetColor(cfg.Theme.BackgroundColor))).
		SetClipboard(func(s string) {
			_ = clipboard.WriteAll(s)
		}, func() string {
			text, _ := clipboard.ReadAll()
			return text
		}).
		SetInputCapture(mi.onInputCapture)

	mi.autocomplete.Box = ui.NewConfiguredBox(mi.autocomplete.Box, &mi.cfg.Theme)
	mi.autocomplete.SetTitle("Mention")
	mi.autocomplete.
		ShowSecondaryText(false).
		SetSelectedStyle(tcell.StyleDefault.
			Background(tcell.ColorWhite).
			Foreground(tcell.ColorBlack))
	mi.autocomplete.SetRect(0, 0, 0, 0)
	b := mi.autocomplete.GetBorderSet()
	b.BottomLeft = b.BottomT
	b.BottomRight = b.BottomT
	mi.autocomplete.SetBorderSet(b)
	return mi
}

func (mi *messageInput) reset() {
	mi.replyMessageID = 0
	mi.SetTitle("")
	mi.SetText("", true)
}

func (mi *messageInput) onInputCapture(event *tcell.EventKey) *tcell.EventKey {
	switch event.Name() {
	case mi.cfg.Keys.MessageInput.Send:
		if app.pages.GetVisible(app.autocompletePage) {
			mi.tabComplete(false)
			return nil
		}
		mi.send()
		return nil
	case mi.cfg.Keys.MessageInput.Editor:
		mi.stopTabCompletion()
		mi.editor()
		return nil
	case mi.cfg.Keys.MessageInput.Cancel:
		if app.pages.GetVisible(app.autocompletePage) {
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
		if app.pages.GetVisible(app.autocompletePage) {
			count := mi.autocomplete.GetItemCount()
			cur := mi.autocomplete.GetCurrentItem()
			switch event.Name() {
			case mi.cfg.Keys.Autocomplete.Down:
				mi.autocomplete.SetCurrentItem((cur+1) % count)
				return nil
			case mi.cfg.Keys.Autocomplete.Up:
				if cur == 0 {
					cur = count
				}
				mi.autocomplete.SetCurrentItem(cur - 1)
				return nil
			}
		}
		go app.QueueUpdateDraw(func() { mi.tabComplete(true) })
	}

	return event
}

func (mi *messageInput) send() {
	if !app.guildsTree.selectedChannelID.IsValid() {
		return
	}

	text := strings.TrimSpace(mi.GetText())
	if text == "" {
		return
	}

	// Process mentions (there's no shortcut, just parse the entire message
	// as markdown and then expand non-code mentions)
	data := api.SendMessageData{
		Content: processText(app.guildsTree.selectedChannelID, []byte(text)),
	}
	if mi.replyMessageID != 0 {
		data.Reference = &discord.MessageReference{MessageID: mi.replyMessageID}
		data.AllowedMentions = &api.AllowedMentions{RepliedUser: option.False}

		if strings.HasPrefix(mi.GetTitle(), "[@]") {
			data.AllowedMentions.RepliedUser = option.True
		}
	}

	go func() {
		if _, err := discordState.SendMessageComplex(app.guildsTree.selectedChannelID, data); err != nil {
			slog.Error("failed to send message in channel", "channel_id", app.guildsTree.selectedChannelID, "err", err)
		}
	}()

	mi.replyMessageID = 0
	mi.reset()

	app.messagesText.Highlight()
	app.messagesText.ScrollToEnd()
}

func processText(cID discord.ChannelID, src []byte) string {
	// ranges we can expandMentions in them
	var rngs [][2]int
	canMention := true
	n := discordmd.Parse(src)
	ast.Walk(n, func(n ast.Node, enter bool) (ast.WalkStatus, error) {
		switch n := n.(type) {
		case *ast.CodeBlock:
			canMention = !enter
		case *discordmd.Inline:
			if (n.Attr & discordmd.AttrMonospace) != 0 {
				canMention = !enter
			}
		case *ast.Text:
			if canMention {
				rngs = append(rngs, [2]int{n.Segment.Start,
					n.Segment.Stop})
			}
		}
		return ast.WalkContinue, nil
	})
	for _, rng := range rngs {
		src = slices.Replace(src, rng[0], rng[1],
			expandMentions(cID, src[rng[0]:rng[1]])...)
	}
	return string(src)
}

func expandMentions(cID discord.ChannelID, src []byte) []byte {
	return mentionRegex.ReplaceAllFunc(src, func(in []byte) (out []byte) {
		out = in
		name := strings.ToLower(string(in[1:]))
		_ = discordState.MemberStore.Each(app.guildsTree.selectedGuildID, func(m *discord.Member) bool {
			if strings.ToLower(m.User.Username) == name {
				if channelHasUser(cID, m.User.ID) {
					out = []byte(m.User.ID.Mention())
				}
				return true
			}
			return false
		})
		return
	})
}

func (mi *messageInput) tabComplete(isAuto bool) {
	posEnd, name, r := mi.GetWordUnderCursor(isValidUserRune)
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
			mems, err := discordState.Cabinet.Members(gID)
			if err != nil {
				slog.Error("fetching members failed", "err", err)
				return
			}
			res := fuzzy.FindFrom(name, &memberList{
				cfg: mi.cfg,
				members: mems,
			})
			for _, r := range res {
				if channelHasUser(cID, mems[r.Index].User.ID) {
					mi.Replace(pos, posEnd, "@"+mems[r.Index].User.Username+" ")
					return
				}
			}
			return
		}
		if mi.autocomplete.GetItemCount() == 0 {
			return
		}
		_, name = mi.autocomplete.GetItemText(mi.autocomplete.GetCurrentItem())
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
		mi.autocomplete.Clear()
		for _, m := range msgs {
			if shown[m.Author.Username] {
				continue
			}
			shown[m.Author.Username] = true
			discordState.MemberState.RequestMember(gID, m.Author.ID)
			if mem, err := discordState.Cabinet.Member(gID, m.Author.ID); err == nil {
				if mi.addAutocompleteItem(gID, mem) {
					break
				}
			}
		}
	} else {
		mi.searchMember(gID, name)
		mi.autocomplete.Clear()
		mems, err := discordState.Cabinet.Members(gID)
		if err != nil {
			slog.Error("fetching members failed", "err", err)
			return
		}
		res := fuzzy.FindFrom(name, &memberList{
			cfg: mi.cfg,
			members: mems,
		})
		if len(res) > int(mi.cfg.AutocompleteLimit) {
			res = res[:int(mi.cfg.AutocompleteLimit)]
		}
		for _, r := range res {
			if channelHasUser(cID, mems[r.Index].User.ID) &&
				mi.addAutocompleteItem(gID, &mems[r.Index]) {
				break
			}
		}
	}

	if mi.autocomplete.GetItemCount() == 0 {
		mi.stopTabCompletion()
		return
	}

	_, col, _, _ := mi.GetCursor()
	mi.showMentionList(col - 1)
}

func (m *memberList) String(i int) string {
	res := ""
	if m.cfg.Theme.Autocomplete.ShowNicknames {
		res = m.members[i].Nick
	}
	if res == "" {
		if m.members[i].User.DisplayName == "" {
			return m.members[i].User.Username
		}
		res = m.members[i].User.DisplayName
	}
	if m.cfg.Theme.Autocomplete.ShowUsernames {
		return res + " " + m.members[i].User.Username
	}
	return res
}

func (m *memberList) Len() int { return len(m.members) }

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
	// Rate limit on our side because we can't distinguish between a
	// successful search and SearchMember not doing anything becuase of its
	// internal rate limit that we can't detect
	if mi.lastSearch.Add(discordState.MemberState.SearchFrequency).After(time.Now()) {
		return
	}
	mi.lastSearch = time.Now()
	app.messagesText.waitForChunkEvent()
	app.messagesText.setFetchingChunk(true, 0)
	discordState.MemberState.SearchMember(gID, name)
	mi.cache.Create(key, app.messagesText.waitForChunkEvent())
}

func isValidUserRune(x rune) bool {
	return (x >= 'a' && x <= 'z') ||
		(x >= 'A' && x <= 'Z') ||
		(x >= '0' && x <= '9') ||
		x == '_' || x == '.'
}

func (mi *messageInput) showMentionList(col int) {
	borders := 0
	if mi.cfg.Theme.Border.Enabled {
		borders = 1
	}
	l := mi.autocomplete
	x, _, _, _ := mi.GetInnerRect()
	_, y, _, _ := mi.GetRect()
	_, _, maxW, maxH := app.messagesText.GetInnerRect()
	if t := int(mi.cfg.Theme.Autocomplete.MaxHeight); t != 0 {
		maxH = min(maxH, t)
	}
	count := l.GetItemCount() + borders
	h := min(count, maxH) + borders + mi.cfg.Theme.Border.Padding[1]
	y -= h
	w := int(mi.cfg.Theme.Autocomplete.MinWidth)
	if w == 0 {
		w = maxW
	} else {
		for i := range count - 1 {
			t, _ := mi.autocomplete.GetItemText(i)
			w = max(w, tview.TaggedStringWidth(t))
		}
		w = min(w+borders*2, maxW)
		x += min(col, maxW-w)
	}
	l.SetRect(x, y, w, h)
	app.pages.ShowPage(app.autocompletePage)
	app.SetFocus(mi)
}

func (mi *messageInput) addAutocompleteItem(gID discord.GuildID, m *discord.Member) bool {
	username := m.User.Username
	if username == "" {
		return false
	}
	var dname string
	if mi.cfg.Theme.Autocomplete.ShowNicknames && m.Nick != "" {
		dname = m.Nick
	} else {
		dname = m.User.DisplayName
	}
	if dname != "" {
		dname = tview.Escape(dname)
	}
	// this is WAY faster than discordState.MemberColor
	if mi.cfg.Theme.Autocomplete.ShowUsernameColors {
		if c, ok := state.MemberColor(m, func(id discord.RoleID) *discord.Role {
			r, _ := discordState.Cabinet.Role(gID, id)
			return r
		}); ok {
			if dname != "" {
				dname = "[" + c.String() + "]" + dname + "[-]"
			} else {
				username = "[" + c.String() + "]" + username + "[-]"
			}
		}
	}
	// The username overwrite in the case of dname == "" is intended
	if presence, _ := discordState.Cabinet.Presence(gID, m.User.ID); presence == nil || presence.Status == discord.OfflineStatus {
		username = "[::d]" + username + "[::D]"
	}
	if dname != "" {
		if mi.cfg.Theme.Autocomplete.ShowUsernames {
			mi.autocomplete.AddItem(dname+" ("+username+")", m.User.Username, 0, nil)
		} else {
			mi.autocomplete.AddItem(dname, m.User.Username, 0, nil)
		}
	} else {
		mi.autocomplete.AddItem(username, m.User.Username, 0, nil)
	}
	return mi.autocomplete.GetItemCount() > int(mi.cfg.AutocompleteLimit)
}

func (mi *messageInput) stopTabCompletion() {
	if mi.cfg.AutocompleteLimit > 0 {
		app.pages.HidePage(app.autocompletePage)
		mi.autocomplete.Clear()
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

	_, _ = file.WriteString(mi.GetText())

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
