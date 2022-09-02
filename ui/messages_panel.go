package ui

import (
	"github.com/diamondburned/arikawa/v3/discord"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
	lua "github.com/yuin/gopher-lua"
	luar "layeh.com/gopher-luar"
)

type MessagesPanel struct {
	*tview.TextView
	// The index of the currently selected message. A negative index indicates that there is no currently selected message.
	SelectedMessage int

	core *Core
}

func NewMessagesPanel(c *Core) *MessagesPanel {
	mp := &MessagesPanel{
		TextView:        tview.NewTextView(),
		SelectedMessage: -1,

		core: c,
	}

	mp.SetDynamicColors(true)
	mp.SetRegions(true)
	mp.SetWordWrap(true)
	mp.SetInputCapture(mp.onInputCapture)
	mp.SetChangedFunc(func() {
		mp.core.Application.Draw()
	})

	mp.SetTitle("Messages")
	mp.SetTitleAlign(tview.AlignLeft)
	mp.SetBorder(true)
	mp.SetBorderPadding(0, 0, 1, 1)

	return mp
}

func (mp *MessagesPanel) onInputCapture(e *tcell.EventKey) *tcell.EventKey {
	if mp.core.ChannelsTree.SelectedChannel == nil {
		return nil
	}

	// Messages should return messages ordered from latest to earliest.
	ms, err := mp.core.State.Cabinet.Messages(mp.core.ChannelsTree.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return nil
	}

	keysTable, ok := mp.core.Config.State.GetGlobal("keys").(*lua.LTable)
	if !ok {
		keysTable = mp.core.Config.State.NewTable()
	}

	messagesPanel, ok := keysTable.RawGetString("messagesPanel").(*lua.LTable)
	if !ok {
		messagesPanel = mp.core.Config.State.NewTable()
	}

	var fn lua.LValue
	messagesPanel.ForEach(func(k, v lua.LValue) {
		keyTable := v.(*lua.LTable)
		if e.Name() == lua.LVAsString(keyTable.RawGetString("name")) {
			fn = keyTable.RawGetString("action")
		}
	})

	if fn != nil {
		mp.core.Config.State.CallByParam(lua.P{
			Fn:      fn,
			NRet:    1,
			Protect: true,
		}, luar.New(mp.core.Config.State, mp.core), luar.New(mp.core.Config.State, e))
		// Returned value
		ret, ok := mp.core.Config.State.Get(-1).(*lua.LUserData)
		if !ok {
			return e
		}

		// Remove returned value
		mp.core.Config.State.Pop(1)

		ev, ok := ret.Value.(*tcell.EventKey)
		if ok {
			return ev
		}
	}

	// Defaults
	switch e.Name() {
	case "Esc":
		mp.SelectedMessage = -1
		mp.core.Application.SetFocus(mp.core.MainFlex)
		mp.
			Clear().
			Highlight().
			SetTitle("")
		return nil
	}

	return e
}

func (mp *MessagesPanel) selectPreviousMessageLua(s *lua.LState) int {
	// Messages should return messages ordered from latest to earliest.
	ms, err := mp.core.State.Cabinet.Messages(mp.core.ChannelsTree.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return returnNilLua(s)
	}

	// If there are no highlighted regions, select the latest (last) message in the messages panel.
	if len(mp.GetHighlights()) == 0 {
		mp.SelectedMessage = 0
	} else {
		// If the selected message is the oldest (first) message, select the latest (last) message in the messages panel.
		if mp.SelectedMessage == len(ms)-1 {
			mp.SelectedMessage = 0
		} else {
			mp.SelectedMessage++
		}
	}

	mp.Highlight(ms[mp.SelectedMessage].ID.String())
	mp.ScrollToHighlight()
	return returnNilLua(s)
}

func (mp *MessagesPanel) selectNextMessageLua(s *lua.LState) int {
	// Messages should return messages ordered from latest to earliest.
	ms, err := mp.core.State.Cabinet.Messages(mp.core.ChannelsTree.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return returnNilLua(s)
	}

	// If there are no highlighted regions, select the latest (last) message in the messages panel.
	if len(mp.GetHighlights()) == 0 {
		mp.SelectedMessage = 0
	} else {
		// If the selected message is the latest (last) message, select the oldest (first) message in the messages panel.
		if mp.SelectedMessage == 0 {
			mp.SelectedMessage = len(ms) - 1
		} else {
			mp.SelectedMessage--
		}
	}

	mp.
		Highlight(ms[mp.SelectedMessage].ID.String()).
		ScrollToHighlight()
	return returnNilLua(s)
}

func (mp *MessagesPanel) selectFirstMessageLua(s *lua.LState) int {
	// Messages should return messages ordered from latest to earliest.
	ms, err := mp.core.State.Cabinet.Messages(mp.core.ChannelsTree.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return returnNilLua(s)
	}

	mp.SelectedMessage = len(ms) - 1
	mp.
		Highlight(ms[mp.SelectedMessage].ID.String()).
		ScrollToHighlight()
	return returnNilLua(s)
}

func (mp *MessagesPanel) selectLastMessageLua(s *lua.LState) int {
	// Messages should return messages ordered from latest to earliest.
	ms, err := mp.core.State.Cabinet.Messages(mp.core.ChannelsTree.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return returnNilLua(s)
	}

	mp.SelectedMessage = 0
	mp.
		Highlight(ms[mp.SelectedMessage].ID.String()).
		ScrollToHighlight()
	return returnNilLua(s)
}

func (mp *MessagesPanel) openMessageActionsListLua(s *lua.LState) int {
	// Messages should return messages ordered from latest to earliest.
	ms, err := mp.core.State.Cabinet.Messages(mp.core.ChannelsTree.SelectedChannel.ID)
	if err != nil || len(ms) == 0 {
		return returnNilLua(s)
	}

	hs := mp.GetHighlights()
	if len(hs) == 0 {
		return returnNilLua(s)
	}

	mID, err := discord.ParseSnowflake(hs[0])
	if err != nil {
		return returnNilLua(s)
	}

	_, m := findMessageByID(ms, discord.MessageID(mID))
	if m == nil {
		return returnNilLua(s)
	}

	actionsList := NewMessageActionsList(mp.core, m)
	mp.core.Application.SetRoot(actionsList, true)
	return returnNilLua(s)
}
