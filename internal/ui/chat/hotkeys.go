package chat

import (
	"fmt"
	"strings"
	"regexp"
	"reflect"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/discordo/internal/markdown"
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v3"
	"github.com/rivo/uniseg"
)

type hotkeysBar struct {
	*tview.TextView
	cfg      *config.Config
	renderer *markdown.InlineRenderer
	hotkeys  []hotkey
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
		renderer: markdown.NewInlineRenderer(),
		cfg:      cfg,
	}
	hkb.TextView.
		SetWrap(false).
		SetBorderPadding(0, 0, cfg.Theme.Hotkeys.Padding[0], cfg.Theme.Hotkeys.Padding[1])
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

// Set hotkeys on focus.
func (b *hotkeysBar) Focus(delegate func(p tview.Primitive)) {
	b._hotkeys()
	b.TextView.Focus(delegate)
}

// Set hotkeys on mouse focus.
func (b *hotkeysBar) MouseHandler() func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
	return b.TextView.WrapMouseHandler(func(action tview.MouseAction, event *tcell.EventMouse, setFocus func(p tview.Primitive)) (consumed bool, capture tview.Primitive) {
		return b.TextView.MouseHandler()(action, event, func(p tview.Primitive) {
			if p == b.TextView {
				b._hotkeys()
			}
			setFocus(p)
		})
	})
}

func (b *hotkeysBar) _hotkeys() {
	cfg := b.cfg.Keybinds
	b.setHotkeys([]hotkey{
		{name: "prev/next", bind: cfg.FocusPrevious + "/" + cfg.FocusNext, hot: true},
		{name: "guilds", bind: cfg.FocusGuildsTree, hot: true},
		{name: "messages", bind: cfg.FocusMessagesList, hot: true},
		{name: "input", bind: cfg.FocusMessageInput, hot: true},
		{name: "channels", bind: cfg.Picker.Toggle, hot: true},
		{name: "logout", bind: cfg.Logout, hot: true},
		{name: "quit", bind: cfg.Quit, hot: true},

	})
}

func (b *hotkeysBar) update() int {
	builder := tview.NewLineBuilder()
	_, _, w, _ := b.GetInnerRect()
	total := 0
	sep := b.renderer.RenderMarkdownLine([]byte(b.cfg.Theme.Hotkeys.Separator), tcell.StyleDefault)
	sepWidth := lineWidth(sep)
	if sepWidth == 0 {
		sep = tview.NewLine(tview.NewSegment(b.cfg.Theme.Hotkeys.Separator, tcell.StyleDefault))
		sepWidth = uniseg.StringWidth(sep[0].Text)
	}
	for _, hk := range b.hotkeys {
		if (!hk.hot && !b.cfg.Theme.Hotkeys.ShowAll) ||
		   (hk.show != nil && !hk.show()) {
			continue
		}
		bind := runePattern.ReplaceAllString(hk.bind, `$1`)
		if b.cfg.Theme.Hotkeys.Compact {
			bind = strings.ReplaceAll(bind, "Ctrl+", "^")
			bind = strings.ReplaceAll(bind, "Shift+", "S-")
			bind = strings.ReplaceAll(bind, "Alt+", "A-")
		}
		line := b.renderer.RenderMarkdownLine(
			[]byte(fmt.Sprintf(b.cfg.Theme.Hotkeys.Format, hk.name, bind)),
			tcell.StyleDefault,
		)
		width := lineWidth(line)
		if total != 0 {
			total += sepWidth
		}
		if total+width >= w {
			total = width
			builder.NewLine()
			appendLine(builder, line)
			continue
		}
		if total != 0 {
			appendLine(builder, sep)
		}
		total += width
		appendLine(builder, line)
	}
	lines := builder.Finish()
	b.SetLines(lines)
	return len(lines)
}

func lineWidth(line tview.Line) int {
	strLen := 0
	for _, segment := range line {
		strLen += uniseg.StringWidth(segment.Text)
	}
	return strLen
}

func appendLine(builder *tview.LineBuilder, line tview.Line) {
	for _, segment := range line {
		builder.Write(segment.Text, segment.Style)
	}
}
