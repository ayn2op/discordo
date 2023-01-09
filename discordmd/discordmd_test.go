package discordmd

import (
	"testing"
)

const input = `**Lorem** ipsum dolor sit amet, consectetur adipiscing __elit.__ Nullam ante magna, luctus in ~~molestie non, elementum sit~~ amet tortor. Nunc euismod urna ac massa dictum ultrices. Donec tempor __dignissim__ ullamcorper. Mauris ultricies, risus non malesuada consectetur, *purus leo interdum purus*, nec vestibulum lacus neque non nulla.`

func TestParse(t *testing.T) {
	const want = `[::b]Lorem[::-] ipsum dolor sit amet, consectetur adipiscing [::u]elit.[::-] Nullam ante magna, luctus in [::s]molestie non, elementum sit[::-] amet tortor. Nunc euismod urna ac massa dictum ultrices. Donec tempor [::u]dignissim[::-] ullamcorper. Mauris ultricies, risus non malesuada consectetur, [::i]purus leo interdum purus[::-], nec vestibulum lacus neque non nulla.`

	got := Parse(input)
	if got != want {
		t.Errorf("got %s; want %s", got, want)
	}
}

func BenchmarkParse(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = Parse(input)
	}
}
