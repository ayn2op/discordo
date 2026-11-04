package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/BurntSushi/toml"
	"github.com/ayn2op/tview/keybind"

	"github.com/ayn2op/discordo/internal/consts"
)

func TestDefaultPath(t *testing.T) {
	t.Run("user config dir fallback", func(t *testing.T) {
		t.Setenv("AppData", "")
		t.Setenv("HOME", "")
		t.Setenv("home", "")
		t.Setenv("XDG_CONFIG_HOME", "")

		// filepath.Join strips the leading dot.
		got := DefaultPath()
		want := filepath.Join(".", consts.Name, fileName)
		if got != want {
			t.Fatalf("got = %v, want = %v", got, want)
		}
	})
}

func TestLoad(t *testing.T) {
	t.Run("invalid config returns error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "bad.toml")
		if err := os.WriteFile(path, []byte("invalid ="), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		if _, err := Load(path); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("valid config does not return error", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "good.toml")
		if err := os.WriteFile(path, []byte("mouse = false\n[help]\nenabled = true"), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		cfg, err := Load(path)
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
		path := filepath.Join(t.TempDir(), "bad-mime.toml")
		if err := os.WriteFile(path, []byte("[ui.messages_list.attachments]\nallowed_mime_types = [\"image/\"]"), os.ModePerm); err != nil {
			t.Fatal(err)
		}

		if _, err := Load(path); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("open with bad path returns error (!= ErrNotExist)", func(t *testing.T) {
		if _, err := Load("bad\x00path"); err == nil {
			t.Fatal("expected error")
		}
	})

	t.Run("missing file uses defaults", func(t *testing.T) {
		path := filepath.Join(t.TempDir(), "missing.toml")
		cfg, err := Load(path)
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

func TestKeybindMarshalTOML(t *testing.T) {
	for _, test := range []struct {
		name string
		keys []string
		want string
	}{
		{"one key", []string{"ctrl+k"}, `"ctrl+k"`},
		{"several keys", []string{"alt+1", "f1"}, `["alt+1","f1"]`},
		{"unbound", nil, `""`},
		// The generated docs are read by people, and json escapes & as \u0026 unless told not to.
		{"a key shift makes of a digit", []string{"alt+&"}, `"alt+&"`},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, err := Keybind{keybind.New(test.keys...)}.MarshalTOML()
			if err != nil {
				t.Fatal(err)
			}

			if string(got) != test.want {
				t.Fatalf("got = %s, want = %s", got, test.want)
			}
		})
	}
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
