package mentionslist

import (
	"github.com/ayn2op/tview/list"
	"github.com/ayn2op/tview/viewport"
)

// Msg is implemented by the messages that move or scroll the list.
type Msg interface{ mentionsList() }

type listMsg list.Change

func (listMsg) mentionsList() {}

type scrollMsg viewport.Change

func (scrollMsg) mentionsList() {}
