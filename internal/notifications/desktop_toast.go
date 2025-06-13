//go:build !darwin

package notifications

import "github.com/gen2brain/beeep"

func sendDesktopNotification(title string, body string, image string, playSound bool, duration int) error {
	if err := beeep.Notify(title, body, image); err != nil {
		return err
	}

	if playSound {
		return beeep.Beep(beeep.DefaultFreq, duration)
	}

	return nil
}
