package form

import "charm.land/bubbles/v2/key"

type KeyMap struct {
	Previous key.Binding
	Next     key.Binding
}

func DefaultKeyMap() KeyMap {
	return KeyMap{
		Previous: key.NewBinding(key.WithKeys("shift+tab")),
		Next:     key.NewBinding(key.WithKeys("tab")),
	}
}
