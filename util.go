package main

func findIndexByMessageID(mID string) int {
	for i, m := range selectedChannel.Messages {
		if m.ID == mID {
			return i
		}
	}

	return -1
}
