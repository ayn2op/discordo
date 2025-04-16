//go:build darwin

package notifications

import (
	gosxnotifier "github.com/deckarep/gosx-notifier"
)

func sendDesktopNotification(title string, body string, image string, playSound bool) error {
	notify := gosxnotifier.NewNotification(body)
	notify.Title = title
	notify.ContentImage = image

	if playSound {
		notify.Sound = gosxnotifier.Ping
	}

	return notify.Push()
}
