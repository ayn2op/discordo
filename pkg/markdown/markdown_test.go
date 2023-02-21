package markdown

import (
	"testing"
)

func TestParse(t *testing.T) {
	testcases := []struct{ input, want string }{
		// Bold
		{"Don't **communicate** by sharing memory, share memory by communicating.", "Don't [::b]communicate[::-] by sharing memory, share memory by communicating."},
		// Italic
		{"*Concurrency* is not parallelism.", "[::i]Concurrency[::-] is not parallelism."},
		// Underline
		{"Channels __orchestrate__; mutexes __serialize__.", "Channels [::u]orchestrate[::-]; mutexes [::u]serialize[::-]."},
		// Strikethrough
		{"~~Cgo~~ is not Go.", "[::s]Cgo[::-] is not Go."},
		// Codeblock
		{"Don't just check `errors`, handle them `gracefully`.", "Don't just check [::r]errors[::-], handle them [::r]gracefully[::-]."},
	}

	for _, testcase := range testcases {
		if got := Parse(testcase.input); got != testcase.want {
			t.Errorf("got %s; want %s", got, testcase.want)
		}
	}
}

func BenchmarkParse(b *testing.B) {
	const input = `**Lorem** ipsum dolor sit amet, consectetur adipiscing __elit.__ Nullam ante magna, luctus in ~~molestie non, elementum sit~~ amet tortor. Nunc euismod urna ac massa dictum ultrices. Donec tempor __dignissim__ ullamcorper. Mauris ultricies, risus non malesuada consectetur, *purus leo interdum purus*, nec vestibulum lacus neque non nulla.`

	for i := 0; i < b.N; i++ {
		_ = Parse(input)
	}
}
