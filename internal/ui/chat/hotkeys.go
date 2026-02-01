package chat

import (
	"fmt"
	"strings"
	"regexp"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview"
)

type hotkeysBar struct {
	*tview.TextView
	cfg     *config.Config
	hotkeys []hotkey
}

type hotkey struct {
	name string      // Hotkey name. E.g., "next"
	bind string      // Hotkey actual keybinding. E.g., "j"
	show func() bool // Returns whether it's ok to display this
	                 // hotkey right now. Use nil for "always"
}

var runePattern = regexp.MustCompile(`Rune\[(.)]`)

func newHotkeysBar(cfg *config.Config) *hotkeysBar {
	hkb := &hotkeysBar{
		TextView: tview.NewTextView(),
		cfg:      cfg,
	}
	hkb.TextView.SetDynamicColors(true)
	return hkb
}

func (b *hotkeysBar) setHotkeys(hotkeys []hotkey) *hotkeysBar {
	b.hotkeys = hotkeys
	return b
}

func (b *hotkeysBar) update() {
	result := &strings.Builder{}
	addSep := false
	for _, hk := range b.hotkeys {
		if hk.show != nil && !hk.show() {
			continue
		}
		if addSep {
			result.WriteString(b.cfg.Theme.Hotkeys.Separator)
		} else {
			addSep = true
		}
		bind := tview.Escape(runePattern.ReplaceAllString(hk.bind, `$1`))
		if b.cfg.Theme.Hotkeys.Compact {
			bind = strings.ReplaceAll(bind, "Ctrl+", "^")
			bind = strings.ReplaceAll(bind, "Shift+", "S-")
		}
		fmt.Fprintf(result, b.cfg.Theme.Hotkeys.Format, hk.name, bind)
	}

	b.SetText(result.String())
}
