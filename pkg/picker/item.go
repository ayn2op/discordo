package picker

type Item struct {
	Text       string
	FilterText string
	Reference  any
}

type Items []Item

func (is Items) String(index int) string {
	return is[index].FilterText
}

func (is Items) Len() int {
	return len(is)
}
