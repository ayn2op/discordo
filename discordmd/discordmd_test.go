package discordmd

import (
	"testing"
)

func TestParse(t *testing.T) {
	const input = `**Lorem** ipsum dolor sit amet, consectetur adipiscing __elit.__ Nullam ante magna, luctus in ~~molestie non, elementum sit~~ amet tortor. Nunc euismod urna ac massa dictum ultrices. Donec tempor __dignissim__ ullamcorper. Mauris ultricies, risus non malesuada consectetur, *purus leo interdum purus*, nec vestibulum lacus neque non nulla.`
	const want = `[::b]Lorem[::-] ipsum dolor sit amet, consectetur adipiscing [::u]elit.[::-] Nullam ante magna, luctus in [::s]molestie non, elementum sit[::-] amet tortor. Nunc euismod urna ac massa dictum ultrices. Donec tempor [::u]dignissim[::-] ullamcorper. Mauris ultricies, risus non malesuada consectetur, [::i]purus leo interdum purus[::-], nec vestibulum lacus neque non nulla.`

	if got := Parse(input); got != want {
		t.Errorf("got %s; want %s", got, want)
	}
}
