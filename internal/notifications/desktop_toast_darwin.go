//go:build darwin

package notifications

import (
	gosxnotifier "github.com/deckarep/gosx-notifier"
)

func sendDesktopNotification(title string, message string, image string, playSound bool, _ int) error {
	notification := gosxnotifier.NewNotification(message)
	notification.Title = title
	notification.ContentImage = image

	if playSound {
		notification.Sound = gosxnotifier.Default
	}

	return notification.Push()
}
