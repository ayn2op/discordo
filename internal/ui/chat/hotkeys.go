package chat

import (
	"fmt"
	"strings"
	"regexp"
	"reflect"

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
	show func() bool // Returns whether it's ok to display this hotkey right now. Use nil for "always"
	hot  bool        // False = only show if theme.hotkeys.show_all is set
}

var runePattern = regexp.MustCompile(`Rune\[(.)]`)

func newHotkeysBar(cfg *config.Config) *hotkeysBar {
	hkb := &hotkeysBar{
		TextView: tview.NewTextView(),
		cfg:      cfg,
	}
	hkb.TextView.
		SetDynamicColors(true).
		SetWrap(false)
	return hkb
}

func (b *hotkeysBar) hotkeysFromValue(v reflect.Value, showFuncs map[string]func() bool) *hotkeysBar {
	b.hotkeys = []hotkey{}
	return b.appendHotkeysFromValue(v, showFuncs)
}

func (b *hotkeysBar) appendHotkeysFromValue(v reflect.Value, showFuncs map[string]func() bool) *hotkeysBar {
	var show func() bool
	t := v.Type()
	for i := 0; i < t.NumField(); i++ {
		fld := t.Field(i)
		if fld.Type.Kind() == reflect.Struct {
			b.appendHotkeysFromValue(v.Field(i), showFuncs)
			continue
		}
		name := fld.Tag.Get("name")
		bind := v.Field(i).String()
		for fld.Tag.Get("join") == "next" {
			i++
			fld = t.Field(i)
			name = fld.Tag.Get("name")
			bind += "/" + v.Field(i).String()
		}
		if name == "" {
			continue
		}
		if showFuncs != nil {
			show = showFuncs[name]
		} else {
			show = nil
		}
		b.hotkeys = append(b.hotkeys, hotkey{
			name: name,
			bind: bind,
			show: show,
			hot:  fld.Tag.Get("hot") == "true",
		})
	}
	return b
}

func (b *hotkeysBar) setHotkeys(hks []hotkey) *hotkeysBar {
	b.hotkeys = hks
	return b
}

func (b *hotkeysBar) appendHotkeys(hks []hotkey) *hotkeysBar {
	b.hotkeys = append(b.hotkeys, hks...)
	return b
}

func (b *hotkeysBar) update() int {
	result := strings.Builder{}
	_, _, w, _ := b.GetRect()
	lines := 1
	total := 0
	strLen := 0
	sep := b.cfg.Theme.Hotkeys.Separator
	sepLen := tview.TaggedStringWidth(sep)
	for i := range b.hotkeys {
		hk := b.hotkeys[i]
		if (!hk.hot && !b.cfg.Theme.Hotkeys.ShowAll) ||
		   (hk.show != nil && !hk.show()) {
			continue
		}
		bind := tview.Escape(runePattern.ReplaceAllString(hk.bind, `$1`))
		if b.cfg.Theme.Hotkeys.Compact {
			bind = strings.ReplaceAll(bind, "Ctrl+", "^")
			bind = strings.ReplaceAll(bind, "Shift+", "S-")
		}
		str := fmt.Sprintf(b.cfg.Theme.Hotkeys.Format, hk.name, bind)
		strLen = tview.TaggedStringWidth(str)
		total += strLen
		if i != 0 {
			total += sepLen
		}
		if total >= w {
			total = strLen
			lines++
			result.WriteRune('\n')
			result.WriteString(str)
			continue
		}
		if i != 0 {
			result.WriteString(sep)
		}
		result.WriteString(str)
	}
	b.SetText(result.String())
	return lines
}
