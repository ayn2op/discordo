package autocomplete

import (
	"github.com/ayn2op/tview/viewport"
)

// SetItemsMsg replaces the items; none hides the list.
type SetItemsMsg struct {
	Title string
	Items []Item
}

// Msg is a message for the list.
type Msg interface{ autocomplete() }

type listMsg int

func (listMsg) autocomplete() {}

type scrollMsg viewport.ScrollState

func (scrollMsg) autocomplete() {}
