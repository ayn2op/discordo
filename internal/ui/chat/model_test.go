package chat

import (
	"fmt"
	"image"
	"path/filepath"
	"testing"
	"time"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/ayn2op/arikawa/v3/gateway"
	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/ui/images"
	"github.com/ayn2op/tview"
	tviewimage "github.com/ayn2op/tview/image"
	"github.com/gdamore/tcell/v3"
)

func newTestModel(t *testing.T) *Model {
	t.Helper()
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg, "")
	return &m
}

func TestModelView(t *testing.T) {
	t.Run("typing in the channels picker", func(t *testing.T) {
		m := newTestModel(t)
		m.openPicker()
		area := tview.Rectangle{Width: 120, Height: 40}
		// The picker's action passes through the panes below it, which must leave it alone.
		msg := m.View().Handle(tcell.NewEventKey(tcell.KeyRune, "a", tcell.ModNone), area)
		if msg == nil {
			t.Fatal("typing produced no message")
		}
		if _, ok := msg.(paneMsg); ok {
			t.Fatalf("the picker's action was marked for a pane: %v", msg)
		}
		*m, _ = m.Update(msg)
	})
}

func TestModelUpdate(t *testing.T) {
	t.Run("typing expires unless extended", func(t *testing.T) {
		m := newTestModel(t)
		m.addTyper(1)
		m.addTyper(1)
		*m, _ = m.Update(typingExpiredMsg{userID: 1})
		if _, ok := m.typers[1]; !ok {
			t.Fatal("an extended typing indicator expired early")
		}
		m.typers[1] = time.Now().Add(-time.Second)
		*m, _ = m.Update(typingExpiredMsg{userID: 1})
		if _, ok := m.typers[1]; ok {
			t.Fatal("the typing indicator did not expire")
		}
	})
}

func TestModelAddMessageOrNotify(t *testing.T) {
	for _, tt := range []struct {
		name          string
		focused       bool
		whenUnfocused bool
		want          bool
	}{
		{"focused", true, true, false},
		{"unfocused", false, true, true},
		{"unfocused but disabled", false, false, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m := newTestModel(t)
			m.cfg.Notifications.WhenUnfocused = tt.whenUnfocused
			// Loading an avatar is a command too.
			m.cfg.UI.MessagesList.Avatars.Preview = false
			m.setSelectedChannel(&discord.Channel{ID: 1})
			*m, _ = m.Update(tcell.NewEventFocus(tt.focused))
			cmd := m.addMessageOrNotify(&gateway.MessageCreateEvent{ID: 1, ChannelID: 1})
			if got := cmd != nil; got != tt.want {
				t.Fatalf("notified = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModelCycleFocus(t *testing.T) {
	m := newTestModel(t)
	m.setSelectedChannel(&discord.Channel{ID: 1, GuildID: 2, Type: discord.GuildText})
	m.focused = paneGuildsTree

	// The composer is disabled without permission to send messages, so it is skipped.
	for _, test := range []struct {
		name  string
		step  pane
		panes []pane
	}{
		{"forward", 1, []pane{paneMessagesList, paneMembersTree, paneGuildsTree}},
		{"backward", -1, []pane{paneMembersTree, paneMessagesList, paneGuildsTree}},
	} {
		t.Run(test.name, func(t *testing.T) {
			m.focused = paneGuildsTree
			for i, want := range test.panes {
				t.Run(fmt.Sprintf("step %d", i+1), func(t *testing.T) {
					if m.cycleFocus(test.step); m.focused != want {
						t.Fatalf("focused = %v, want %v", m.focused, want)
					}
				})
			}
		})
	}
}

func TestModelReloadImages(t *testing.T) {
	const url = "avatar.png"
	m := newTestModel(t)
	m.kitty, m.cellWidth, m.cellHeight = true, 9, 18
	m.images[url] = tviewimage.New(image.NewAlpha(image.Rect(0, 0, 1, 1)))
	cell := m.imageCell()

	t.Run("same cell size", func(t *testing.T) {
		m.reloadImages(cell)
		if len(m.images) != 1 {
			t.Fatal("images were dropped though the cell size did not change")
		}
	})

	t.Run("new cell size", func(t *testing.T) {
		m.cellWidth, m.cellHeight = 25, 49
		m.reloadImages(cell)
		if len(m.images) != 0 {
			t.Fatal("images sized for the old cell size were kept")
		}
	})

	t.Run("stale download", func(t *testing.T) {
		m.images[url] = tviewimage.New(image.NewAlpha(image.Rect(0, 0, 1, 1)))
		// Only an accepted image is sent to kitty.
		*m, _ = m.Update(images.LoadedMsg{URL: url, Image: m.images[url], Cell: cell})
		if m.kittyImages != 0 {
			t.Fatal("an image sized for the old cell size was shown")
		}

		*m, _ = m.Update(images.LoadedMsg{URL: url, Image: m.images[url], Cell: m.imageCell()})
		if m.kittyImages != 1 {
			t.Fatal("an image sized for the cell size was not shown")
		}
	})
}
