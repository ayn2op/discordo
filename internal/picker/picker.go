package picker

import (
	"github.com/rivo/tview"
	"github.com/sahilm/fuzzy"
)

type Picker struct {
	*tview.Flex
	Input *tview.InputField
	List  *tview.List

	app   *tview.Application
	items Items
}

func New(app *tview.Application, items Items) *Picker {
	p := &Picker{
		Flex:  tview.NewFlex(),
		Input: tview.NewInputField(),
		List:  tview.NewList(),

		app:   app,
		items: items,
	}

	// Render all of the items initially.
	p.changed("")

	p.Input.SetChangedFunc(p.changed)
	p.List.ShowSecondaryText(false).SetFocusFunc(func() {
		app.SetFocus(p.Input)
	})
	p.Flex.
		SetDirection(tview.FlexRow).
		AddItem(p.Input, 3, 1, true).
		AddItem(p.List, 0, 1, false)
	return p
}

func (p *Picker) changed(text string) {
	var fuzzied Items
	if text == "" {
		fuzzied = append(fuzzied, p.items...)
	} else {
		matches := fuzzy.FindFrom(text, p.items)
		for _, match := range matches {
			fuzzied = append(fuzzied, p.items[match.Index])
		}
	}

	p.List.Clear()

	for _, item := range fuzzied {
		p.List.AddItem(item.text, "", 0, item.selected)
	}
}
