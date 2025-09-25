//go:build darwin

package notifications

import (
	gosxnotifier "github.com/deckarep/gosx-notifier"
)

func sendDesktopNotification(title string, message string, image string, playSound bool, _ int) error {
	n := gosxnotifier.NewNotification(message)
	n.Title = title
	n.ContentImage = image

	if playSound {
		n.Sound = gosxnotifier.Default
	}

	return n.Push()
}
