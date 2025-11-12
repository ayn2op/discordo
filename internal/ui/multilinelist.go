package ui

import (
	"strconv"
	"github.com/ayn2op/tview"
)

type ListItem struct {
	Text          string
	HoldValue     any
	HighlightFunc func(itme *ListItem, highlighted bool)
}

type MultiLineList struct {
	*tview.TextView
	items            []ListItem
	highlightedIndex int
}

func NewMultiLineList() *MultiLineList {
	return &MultiLineList{
		TextView:         tview.NewTextView().
		                  SetDynamicColors(true).
		                  SetRegions(true).
		                  SetWordWrap(true),
		highlightedIndex: -1,
	}
}

func (mll *MultiLineList) DrawItems() {
	w := mll.TextView.BatchWriter()
	w.Clear()
	lastIdx := len(mll.items)-1
	for idx, item := range mll.items {
		w.Write([]byte{'[', '"'})
		w.Write([]byte(strconv.Itoa(idx)))
		w.Write([]byte{'"', ']'})
		w.Write([]byte(item.Text))
		w.Write([]byte{'[', '"', '"', ']'})
		if idx != lastIdx {
			w.Write([]byte{'\n'})
		}
	}
	w.Close()
}

func (mll *MultiLineList) Highlight(i int) *MultiLineList {
	oldIdx := mll.highlightedIndex
	if oldIdx > mll.highlightedIndex {
		oldIdx = -1
		mll.highlightedIndex = -1
	}
	if i >= len(mll.items) {
		i = -1
	}
	mll.highlightedIndex = i
	if oldIdx > 0 && mll.items[oldIdx].HighlightFunc != nil {
		mll.items[oldIdx].HighlightFunc(&mll.items[oldIdx], false)
	}
	if i > 0 && mll.items[i].HighlightFunc != nil {
		mll.items[i].HighlightFunc(&mll.items[i], true)
	}
	if mll.highlightedIndex < 0 {
		mll.TextView.Highlight("")
	} else {
		mll.TextView.Highlight(strconv.Itoa(mll.highlightedIndex))
		mll.TextView.ScrollToHighlight()
	}
	return mll
}

func (mll *MultiLineList) GetHighlightedIndex() int {
	return mll.highlightedIndex
}

func (mll *MultiLineList) Clear() *MultiLineList {
	mll.items = nil
	mll.highlightedIndex = -1
	mll.TextView.Clear()
	return mll
}

func (mll *MultiLineList) AppendItem(text string, holdValue any, highlightFunc func(itme *ListItem, highlighted bool)) *MultiLineList {
	mll.items = append(mll.items, ListItem {
		Text: text,
		HoldValue: holdValue,
		HighlightFunc: highlightFunc,
	})
	return mll
}

func (mll *MultiLineList) GetItem(i int) ListItem {
	return mll.items[i]
}

func (mll *MultiLineList) GetHighlightedItem() ListItem {
	return mll.items[mll.highlightedIndex]
}

func (mll *MultiLineList) GetItemCount() int {
	return len(mll.items)
}
