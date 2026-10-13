package mentionslist

import (
	"github.com/ayn2op/tview/viewport"
)

// Msg is implemented by the messages that move or scroll the list.
type Msg interface{ mentionsList() }

// listMsg is the item to move the cursor to.
type listMsg int

func (listMsg) mentionsList() {}

type scrollMsg viewport.ScrollState

func (scrollMsg) mentionsList() {}
