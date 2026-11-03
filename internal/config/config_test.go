package config

import (
	"maps"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"

	"github.com/ayn2op/discordo/internal/consts"
)

func TestDefaultDir(t *testing.T) {
	t.Run("user config dir fallback", func(t *testing.T) {
		t.Setenv("AppData", "")
		t.Setenv("HOME", "")
		t.Setenv("home", "")
		t.Setenv("XDG_CONFIG_HOME", "")

		// filepath.Join strips the leading dot.
		got := DefaultDir()
		want := filepath.Join(".", consts.Name)
		if got != want {
			t.Fatalf("got = %v, want = %v", got, want)
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("theme files are based on the built-in default", func(t *testing.T) {
		saved := maps.Clone(themes)
		t.Cleanup(func() { themes = saved })

		dir := filepath.Join(t.TempDir(), "themes")
		if err := os.Mkdir(dir, os.ModePerm); err != nil {
			t.Fatal(err)
		}
		for name, content := range map[string]string{
			"a.toml":       "[primary.base]\nbackground = \"#00ff00\"",
			"default.toml": "[background.base]\nbackground = \"#ff0000\"",
			"z.toml":       "[primary.base]\nbackground = \"#00ff00\"",
		} {
			if err := os.WriteFile(filepath.Join(dir, name), []byte(content), os.ModePerm); err != nil {
				t.Fatal(err)
			}
		}

		if _, err := Load(filepath.Dir(dir)); err != nil {
			t.Fatal(err)
		}
		if a, z := themes["a"].Background.Base, themes["z"].Background.Base; a != z {
			t.Fatalf("a's base = %v, z's base = %v, want the same", a.GetBackground(), z.GetBackground())
		}
	})

	t.Run("invalid config returns error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, fileName), []byte("invalid ="), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		if _, err := Load(dir); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("valid config does not return error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, fileName), []byte("mouse = false\n[help]\nenabled = true"), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(dir)
		if err != nil {
			t.Fatal(err)
		}

		if cfg.Mouse != false {
			t.Fatalf("got = %v, want = false", cfg.Mouse)
		}
		if !cfg.Help.Enabled {
			t.Fatal("help enabled = false, want true")
		}
	})

	t.Run("invalid allowed MIME type returns error", func(t *testing.T) {
		dir := t.TempDir()
		if err := os.WriteFile(filepath.Join(dir, fileName), []byte("[ui.messages_list.attachments]\nallowed_mime_types = [\"image/\"]"), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		if _, err := Load(dir); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("open with bad path returns error (!= ErrNotExist)", func(t *testing.T) {
		if _, err := Load("bad\x00path"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing file uses defaults", func(t *testing.T) {
		dir := t.TempDir()
		cfg, err := Load(dir)
		if err != nil {
			t.Fatal(err)
		}

		defCfg := Default()
		defCfg.applyDefaults()

		if !reflect.DeepEqual(defCfg, *cfg) {
			t.Fatalf("got = %+v, want = %+v", *cfg, defCfg)
		}
	})
}

// The generated docs show defaults as they are marshaled, so those must parse back to the same defaults.
func TestDefault(t *testing.T) {
	t.Run("round trips through TOML", func(t *testing.T) {
		data, err := toml.Marshal(Default())
		if err != nil {
			t.Fatal(err)
		}
		// Help descriptions are not part of the TOML.
		got := Config{Keybinds: defaultKeybinds()}
		if err := toml.Unmarshal(data, &got); err != nil {
			t.Fatal(err)
		}
		if want := Default(); !reflect.DeepEqual(got, want) {
			t.Fatalf("got = %+v, want = %+v", got, want)
		}
	})
}

func TestMIMETypesHas(t *testing.T) {
	tests := []struct {
		name        string
		allowed     MIMETypes
		mediaType   string
		wantAllowed bool
	}{
		{"exact", []string{"application/pdf"}, "application/pdf", true},
		{"exact mismatch", []string{"application/pdf"}, "application/zip", false},
		{"subtype wildcard", []string{"image/*"}, "image/png", true},
		{"wildcard mismatch", []string{"image/*"}, "video/mp4", false},
		{"all wildcard", []string{"*/*"}, "video/mp4", true},
		{"case and parameters", []string{"image/*"}, "Image/PNG; charset=utf-8", true},
		{"invalid content type", []string{"*/*"}, "not-a-mime-type", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.allowed.Has(tt.mediaType); got != tt.wantAllowed {
				t.Fatalf("Has(%q) = %v, want %v", tt.mediaType, got, tt.wantAllowed)
			}
		})
	}
}
