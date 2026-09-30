package login

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/tview/keybind"
)

func TestModelShortHelp(t *testing.T) {
	cfg, err := config.Load(filepath.Join(t.TempDir(), "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	m := NewModel(cfg)
	for _, tt := range []struct {
		name               string
		active             tab
		wantPrev, wantNext bool
	}{
		{"first tab", tabPassword, false, true},
		{"middle tab", tabQR, true, true},
		{"last tab", tabToken, true, false},
	} {
		t.Run(tt.name, func(t *testing.T) {
			m.active = tt.active
			short := m.ShortHelp()
			has := func(desc string) bool {
				return slices.ContainsFunc(short, func(k keybind.Keybind) bool { return k.Help().Desc == desc })
			}
			if has("prev tab") != tt.wantPrev || has("next tab") != tt.wantNext {
				t.Fatalf("prev = %v, next = %v", has("prev tab"), has("next tab"))
			}
		})
	}
}
