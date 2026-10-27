//go:build !windows

package tzlocal

import (
	"errors"
	"os"
	"strings"
)

// zoneinfoDirs are where /etc/localtime may link into.
var zoneinfoDirs = []string{"/usr/share/zoneinfo/", "../usr/share/zoneinfo/", "/etc/zoneinfo/", "../etc/zoneinfo/", "/var/db/timezone/zoneinfo/"}

func system() (string, error) {
	return fromFiles("/etc/localtime", "/etc/timezone")
}

// fromFiles returns the zone localtime links to, or else the one timezone names.
func fromFiles(localtime, timezone string) (string, error) {
	target, err := os.Readlink(localtime)
	if err == nil {
		if name, ok := zoneFromLink(target); ok {
			return name, nil
		}
		err = errors.New(localtime + " links outside the zoneinfo directory: " + target)
	}

	contents, timezoneErr := os.ReadFile(timezone)
	if timezoneErr != nil {
		return "", errors.Join(err, timezoneErr)
	}
	return strings.TrimSpace(string(contents)), nil
}

func zoneFromLink(target string) (string, bool) {
	for _, dir := range zoneinfoDirs {
		if name, ok := strings.CutPrefix(target, dir); ok {
			return name, true
		}
	}
	return "", false
}
