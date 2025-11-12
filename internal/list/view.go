package list

import (
	"github.com/ayn2op/tview"
	"github.com/gdamore/tcell/v2"
)

type Item interface {
	tview.Primitive
	Height() int
}

type View struct {
	*tview.Box
	items    []Item
	selected int
}

func NewView() *View {
	return &View{
		Box: tview.NewBox(),
	}
}

func (v *View) Clear() {
	v.items = nil
}

func (v *View) Append(item Item) {
	v.items = append(v.items, item)
}

func (v *View) Draw(screen tcell.Screen) {
	v.DrawForSubclass(screen, v)

	x, y, width, height := v.GetInnerRect()
	bottom := height + y

	for _, item := range v.items[v.selected:] {
		itemHeight := item.Height()
		// stop when exceeding view height
		if y+itemHeight > bottom {
			break
		}

		item.SetRect(x, y, width, itemHeight)
		item.Draw(screen)
		y += itemHeight
	}
}

func (v *View) InputHandler() func(*tcell.EventKey, func(tview.Primitive)) {
	return v.WrapInputHandler(func(event *tcell.EventKey, f func(tview.Primitive)) {
		switch event.Key() {
		case tcell.KeyHome:
			v.selectFirst()
		case tcell.KeyEnd:
			v.selectLast()
		case tcell.KeyUp:
			v.selectPrevious()
		case tcell.KeyDown:
			v.selectNext()

		}
	})
}

func (v *View) calculateHeight(item *tview.TextView) int {
	count := item.GetWrappedLineCount()
	borders := item.GetBorders()
	if borders.Has(tview.BordersAll) {
		count += 2
	}

	return count
}

func (v *View) selectFirst() {
	v.selected = 0
}

func (v *View) selectLast() {
	count := len(v.items)
	if count == 0 {
		return
	}

	v.selected = count - 1
}

func (v *View) selectPrevious() {
	v.selected = max(v.selected-1, 0)
}

func (v *View) selectNext() {
	count := len(v.items)
	if count == 0 {
		return
	}

	v.selected = min(v.selected+1, count-1)
}
