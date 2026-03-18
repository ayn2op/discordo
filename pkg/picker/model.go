package picker

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/flex"
	"github.com/ayn2op/tview/keybind"
	"github.com/ayn2op/tview/list"
	"github.com/gdamore/tcell/v3"
	"github.com/sahilm/fuzzy"
)

// bottom border + value
const inputHeight = 2

type Model struct {
	*flex.Model
	input *tview.InputField
	list  *list.Model

	keyMap *KeyMap

	items    Items
	filtered Items
}

func NewModel() *Model {
	m := &Model{
		Model: flex.NewModel(),
		input: tview.NewInputField(),
		list:  list.NewModel(),
	}

	// Show a horizontal bottom border to visually separate input from list.
	var borderSet tview.BorderSet
	borderSet.Bottom = tview.BoxDrawingsLightHorizontal
	borderSet.BottomLeft = borderSet.Bottom
	borderSet.BottomRight = borderSet.Bottom

	m.input.
		SetChangedFunc(m.onInputChanged).
		SetLabel("> ").
		SetBorders(tview.BordersBottom).
		SetBorderSet(borderSet).
		SetBorderStyle(tcell.StyleDefault.Dim(true))

	m.
		SetDirection(flex.DirectionRow).
		AddItem(m.input, inputHeight, 0, true).
		AddItem(m.list, 0, 1, false)

	m.Update()
	return m
}

func (m *Model) setFilteredItems(filtered Items) {
	m.filtered = filtered

	m.list.SetBuilder(func(index int, cursor int) list.Item {
		if index < 0 || index >= len(m.filtered) {
			return nil
		}
		style := tcell.StyleDefault
		if index == cursor {
			style = style.Reverse(true)
		}
		return tview.NewTextView().
			SetScrollable(false).
			SetWrap(false).
			SetWordWrap(false).
			SetTextStyle(style).
			SetLines([]tview.Line{{{Text: m.filtered[index].Text, Style: style}}})
	})

	if len(filtered) == 0 {
		m.list.SetCursor(-1)
	} else {
		m.list.SetCursor(0)
	}
}

func (m *Model) SetKeyMap(keyMap *KeyMap) {
	m.keyMap = keyMap
}

// SetScrollBarVisibility sets when the picker's list scrollBar is rendered.
func (m *Model) SetScrollBarVisibility(visibility list.ScrollBarVisibility) {
	m.list.SetScrollBarVisibility(visibility)
}

// SetScrollBar sets the scrollBar primitive used by the picker's list.
func (m *Model) SetScrollBar(scrollBar *tview.ScrollBar) {
	m.list.SetScrollBar(scrollBar)
}

func (m *Model) ClearInput() {
	m.input.SetText("")
}

func (m *Model) ClearList() {
	m.filtered = nil
	m.list.Clear()
}

func (m *Model) ClearItems() {
	m.items = nil
	m.filtered = nil
}

func (m *Model) AddItem(item Item) {
	m.items = append(m.items, item)
}

func (m *Model) Update() {
	m.ClearInput()
	m.onInputChanged("")
}

func (m *Model) onInputChanged(text string) {
	var fuzzied Items
	if text == "" {
		fuzzied = append(fuzzied, m.items...)
	} else {
		matches := fuzzy.FindFrom(text, m.items)
		for _, match := range matches {
			fuzzied = append(fuzzied, m.items[match.Index])
		}
	}
	m.setFilteredItems(fuzzied)
}

func (m *Model) HandleEvent(event tview.Event) tview.Command {
	switch event := event.(type) {
	case *tview.KeyEvent:
		if m.keyMap != nil {
			switch {
			case keybind.Matches(event, m.keyMap.Up):
				m.list.HandleEvent(tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone))
				return nil
			case keybind.Matches(event, m.keyMap.Down):
				m.list.HandleEvent(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
				return nil
			case keybind.Matches(event, m.keyMap.Top):
				m.list.HandleEvent(tcell.NewEventKey(tcell.KeyHome, "", tcell.ModNone))
				return nil
			case keybind.Matches(event, m.keyMap.Bottom):
				m.list.HandleEvent(tcell.NewEventKey(tcell.KeyEnd, "", tcell.ModNone))
				return nil

			case keybind.Matches(event, m.keyMap.Select):
				return m._select()
			case keybind.Matches(event, m.keyMap.Cancel):
				return cancel()
			}
		}

		return m.Model.HandleEvent(event)
	}
	return m.Model.HandleEvent(event)
}
