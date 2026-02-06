package tabs

type Keybinds struct {
	Previous string
	Next     string
}

func DefaultKeybinds() Keybinds {
	return Keybinds{
		Previous: "Ctrl+H",
		Next:     "Ctrl+L",
	}
}
