package chat

import (
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
	sep      tview.Line
	sepWidth int
}

type hotkey struct {
	name string
	bind string
	show func() bool // Returns whether it's ok to display this hotkey right now. Use nil for "always"
	hot  bool        // False = only show if theme.hotkeys.show_all is set

	line  tview.Line
	width int
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
		SetBorderPadding(0, 0, cfg.Theme.HotkeysBar.Padding[0], cfg.Theme.HotkeysBar.Padding[1])
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
	b.process()
	return b
}

func (b *hotkeysBar) setHotkeys(hks []hotkey) *hotkeysBar {
	b.hotkeys = hks
	b.process()
	return b
}

func (b *hotkeysBar) appendHotkeys(hks []hotkey) *hotkeysBar {
	b.hotkeys = append(b.hotkeys, hks...)
	b.process()
	return b
}

func (b *hotkeysBar) process() {
	for i, hk := range b.hotkeys {
		bind := runePattern.ReplaceAllString(hk.bind, `$1`)
		if b.cfg.Theme.HotkeysBar.Compact {
			bind = strings.ReplaceAll(bind, "Ctrl+", "^")
			bind = strings.ReplaceAll(bind, "Shift+", "S-")
			bind = strings.ReplaceAll(bind, "Alt+", "A-")
		}
		b.hotkeys[i].line = b.renderer.RenderMarkdownLine([]byte(b.cfg.Theme.HotkeysBar.Format), tcell.StyleDefault, strings.NewReplacer("{{name}}", hk.name, "{{keybind}}", bind))
		b.hotkeys[i].width = lineWidth(b.hotkeys[i].line)
	}
	b.sep = tview.NewLine(tview.NewSegment(b.cfg.Theme.HotkeysBar.Separator, tcell.StyleDefault))
	b.sepWidth = uniseg.StringWidth(b.cfg.Theme.HotkeysBar.Separator)
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
	for _, hk := range b.hotkeys {
		if (!hk.hot && !b.cfg.Theme.HotkeysBar.ShowAll) ||
		   (hk.show != nil && !hk.show()) {
			continue
		}
		if total != 0 {
			total += b.sepWidth
		}
		if total+hk.width >= w {
			total = hk.width
			builder.NewLine()
			appendLine(builder, hk.line)
			continue
		}
		if total != 0 {
			appendLine(builder, b.sep)
		}
		total += hk.width
		appendLine(builder, hk.line)
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
