package picker

type Item struct {
	text     string
	selected func()
}

func NewItem(text string, selected func()) *Item {
	return &Item{
		text:     text,
		selected: selected,
	}
}

type Items []*Item

func (is Items) String(i int) string {
	return is[i].text
}

func (is Items) Len() int {
	return len(is)
}
