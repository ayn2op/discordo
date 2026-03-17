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

type Picker struct {
	*flex.Model
	input *tview.InputField
	list  *list.Model

	keyMap *KeyMap

	items    Items
	filtered Items
}

func New() *Picker {
	p := &Picker{
		Model: flex.NewModel(),
		input: tview.NewInputField(),
		list:  list.NewModel(),
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
		SetDirection(flex.DirectionRow).
		AddItem(p.input, inputHeight, 0, true).
		AddItem(p.list, 0, 1, false)

	p.Update()
	return p
}

func (p *Picker) setFilteredItems(filtered Items) {
	p.filtered = filtered

	p.list.SetBuilder(func(index int, cursor int) list.Item {
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
func (p *Picker) SetScrollBarVisibility(visibility list.ScrollBarVisibility) {
	p.list.SetScrollBarVisibility(visibility)
}

// SetScrollBar sets the scrollBar primitive used by the picker's list.
func (p *Picker) SetScrollBar(scrollBar *tview.ScrollBar) {
	p.list.SetScrollBar(scrollBar)
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

func (p *Picker) HandleEvent(event tcell.Event) tview.Command {
	switch event := event.(type) {
	case *tview.KeyEvent:
		if p.keyMap != nil {
			switch {
			case keybind.Matches(event, p.keyMap.Up):
				p.list.HandleEvent(tcell.NewEventKey(tcell.KeyUp, "", tcell.ModNone))
				return nil
			case keybind.Matches(event, p.keyMap.Down):
				p.list.HandleEvent(tcell.NewEventKey(tcell.KeyDown, "", tcell.ModNone))
				return nil
			case keybind.Matches(event, p.keyMap.Top):
				p.list.HandleEvent(tcell.NewEventKey(tcell.KeyHome, "", tcell.ModNone))
				return nil
			case keybind.Matches(event, p.keyMap.Bottom):
				p.list.HandleEvent(tcell.NewEventKey(tcell.KeyEnd, "", tcell.ModNone))
				return nil

			case keybind.Matches(event, p.keyMap.Select):
				return p._select()
			case keybind.Matches(event, p.keyMap.Cancel):
				return cancel()
			}
		}

		return p.Model.HandleEvent(event)
	}
	return p.Model.HandleEvent(event)
}
