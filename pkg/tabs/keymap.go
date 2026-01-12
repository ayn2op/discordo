package tabs

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Previous key.Binding
	Next     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Previous: key.NewBinding(key.WithKeys("ctrl+p")),
		Next:     key.NewBinding(key.WithKeys("ctrl+n")),
	}
}
