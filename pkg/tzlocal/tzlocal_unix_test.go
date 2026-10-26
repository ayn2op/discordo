//go:build !windows

package tzlocal

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestZoneFromLink(t *testing.T) {
	for _, test := range []struct {
		target string
		want   string
		ok     bool
	}{
		{"/usr/share/zoneinfo/America/New_York", "America/New_York", true},
		{"../usr/share/zoneinfo/Europe/Berlin", "Europe/Berlin", true},
		{"/etc/zoneinfo/Etc/UTC", "Etc/UTC", true},
		{"/var/db/timezone/zoneinfo/Asia/Kolkata", "Asia/Kolkata", true},
		{"/tmp/localtime", "", false},
	} {
		t.Run(test.target, func(t *testing.T) {
			if got, ok := zoneFromLink(test.target); got != test.want || ok != test.ok {
				t.Errorf("got %q, %v, want %q, %v", got, ok, test.want, test.ok)
			}
		})
	}
}

func TestFromFiles(t *testing.T) {
	dir := t.TempDir()
	link := func(t *testing.T, target string) string {
		t.Helper()
		path := filepath.Join(t.TempDir(), "localtime")
		if err := os.Symlink(target, path); err != nil {
			t.Fatal(err)
		}
		return path
	}
	timezone := filepath.Join(dir, "timezone")
	if err := os.WriteFile(timezone, []byte("Asia/Tokyo\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(dir, "missing")

	t.Run("link", func(t *testing.T) {
		if got, err := fromFiles(link(t, "/usr/share/zoneinfo/Europe/Berlin"), missing); err != nil || got != "Europe/Berlin" {
			t.Errorf("got %q, %v, want Europe/Berlin", got, err)
		}
	})
	t.Run("link outside zoneinfo", func(t *testing.T) {
		if got, err := fromFiles(link(t, "/tmp/localtime"), timezone); err != nil || got != "Asia/Tokyo" {
			t.Errorf("got %q, %v, want Asia/Tokyo from the timezone file", got, err)
		}
	})
	t.Run("no link", func(t *testing.T) {
		if got, err := fromFiles(missing, timezone); err != nil || got != "Asia/Tokyo" {
			t.Errorf("got %q, %v, want Asia/Tokyo from the timezone file", got, err)
		}
	})
	t.Run("neither", func(t *testing.T) {
		if _, err := fromFiles(missing, missing); err == nil {
			t.Error("no error")
		}
	})
}

func TestName(t *testing.T) {
	if _, err := os.Lstat("/etc/localtime"); err != nil {
		t.Skip("no /etc/localtime:", err)
	}

	name, err := Name()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := time.LoadLocation(name); err != nil {
		t.Errorf("%q is not a zone: %v", name, err)
	}
}
