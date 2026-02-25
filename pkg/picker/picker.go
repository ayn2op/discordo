package picker

import (
	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/keybind"
	"github.com/gdamore/tcell/v3"
	"github.com/sahilm/fuzzy"
)

type (
	SelectedFunc func(item Item)
	CancelFunc   func()
)

type Picker struct {
	*tview.Flex
	input *tview.InputField
	list  *tview.List

	onSelected SelectedFunc
	onCancel   CancelFunc
	keyMap     *KeyMap

	items    Items
	filtered Items
}

func New() *Picker {
	p := &Picker{
		Flex:  tview.NewFlex(),
		input: tview.NewInputField(),
		list:  tview.NewList(),
	}

	// Show a horizontal bottom border to visually separate input from list.
	var borderSet tview.BorderSet
	borderSet.Bottom = tview.BoxDrawingsLightHorizontal
	borderSet.BottomLeft = borderSet.Bottom
	borderSet.BottomRight = borderSet.Bottom

	p.input.
		SetChangedFunc(p.onInputChanged).
		SetLabel("> ").
		SetBorders(tview.BordersBottom).
		SetBorderSet(borderSet).
		SetBorderStyle(tcell.StyleDefault.Dim(true))

	p.
		SetDirection(tview.FlexRow).
		// bottom border + value
		AddItem(p.input, 2, 0, true).
		AddItem(p.list, 0, 1, false)

	p.Update()
	return p
}

func (p *Picker) setFilteredItems(filtered Items) {
	p.filtered = filtered

	p.list.SetBuilder(func(index int, cursor int) tview.ListItem {
		if index < 0 || index >= len(p.filtered) {
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
			SetLines([]tview.Line{{{Text: p.filtered[index].Text, Style: style}}})
	})

	if len(filtered) == 0 {
		p.list.SetCursor(-1)
	} else {
		p.list.SetCursor(0)
	}
}

func (p *Picker) SetKeyMap(keyMap *KeyMap) {
	p.keyMap = keyMap
}

// SetScrollBarVisibility sets when the picker's list scrollBar is rendered.
func (p *Picker) SetScrollBarVisibility(visibility tview.ScrollBarVisibility) {
	p.list.SetScrollBarVisibility(visibility)
}

// SetScrollBar sets the scrollBar primitive used by the picker's list.
func (p *Picker) SetScrollBar(scrollBar *tview.ScrollBar) {
	p.list.SetScrollBar(scrollBar)
}

func (p *Picker) SetSelectedFunc(onSelected SelectedFunc) {
	p.onSelected = onSelected
}

func (p *Picker) SetCancelFunc(onCancel CancelFunc) {
	p.onCancel = onCancel
}

func (p *Picker) ClearInput() {
	p.input.SetText("")
}

func (p *Picker) ClearList() {
	p.filtered = nil
	p.list.Clear()
}

func (p *Picker) ClearItems() {
	p.items = nil
	p.filtered = nil
}

func (p *Picker) AddItem(item Item) {
	p.items = append(p.items, item)
}

func (p *Picker) Update() {
	p.ClearInput()
	p.onInputChanged("")
}

func (p *Picker) onListSelected(index int) {
	if p.onSelected != nil {
		if index >= 0 && index < len(p.filtered) {
			item := p.filtered[index]
			p.onSelected(item)
		}
	}
}

func (p *Picker) onInputChanged(text string) {
	var fuzzied Items
	if text == "" {
		fuzzied = append(fuzzied, p.items...)
	} else {
		matches := fuzzy.FindFrom(text, p.items)
		for _, match := range matches {
			fuzzied = append(fuzzied, p.items[match.Index])
		}
	}
	p.setFilteredItems(fuzzied)
}

func (p *Picker) handleInput(event *tcell.EventKey) *tcell.EventKey {
	if p.keyMap == nil {
		return event
	}

	switch {
	case keybind.Matches(event, p.keyMap.Up):
		p.list.InputHandler(tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone), nil)
		return nil
	case keybind.Matches(event, p.keyMap.Down):
		p.list.InputHandler(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone), nil)
		return nil
	case keybind.Matches(event, p.keyMap.Top):
		p.list.InputHandler(tcell.NewEventKey(tcell.KeyHome, "", tcell.ModNone), nil)
		return nil
	case keybind.Matches(event, p.keyMap.Bottom):
		p.list.InputHandler(tcell.NewEventKey(tcell.KeyEnd, "", tcell.ModNone), nil)
	case keybind.Matches(event, p.keyMap.Select):
		p.onListSelected(p.list.Cursor())
		return nil

	case keybind.Matches(event, p.keyMap.Cancel):
		if p.onCancel != nil {
			p.onCancel()
		}
		return nil
	}

	return event
}

func (p *Picker) InputHandler(event *tcell.EventKey, setFocus func(p2 tview.Primitive)) {
	event = p.handleInput(event)
	if event == nil {
		return
	}
	p.Flex.InputHandler(event, setFocus)
}
