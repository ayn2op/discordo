package discord

import "testing"

func TestParseMarkdown(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"bold", "**test**", "[::b]test[::-]"},
		{"italic", "*test*", "[::i]test[::-]"},
		{"underline", "__test__", "[::u]test[::-]"},
		{"strikethrough", "~~test~~", "[::s]test[::-]"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := ParseMarkdown(test.in); got != test.want {
				t.Errorf("got: %s\nwant: %s", got, test.want)
			}
		})
	}
}

func BenchmarkParseMarkdown(b *testing.B) {
	in := `**Porro mollitia aut odio dolor rerum.** Saepe qui aut reiciendis illo nisi. Id illo et quo consequatur sint labore placeat maiores. __Commodi odio quae reprehenderit.__ Beatae illum est fugiat ut architecto itaque eveniet aut. ~~Consequuntur quas explicabo et impedit eum porro facere et.~~
	
	Sit commodi sed iure et sed quae eveniet. *Sit non distinctio nihil sunt. Nesciunt cumque aspernatur *nulla* porro et earum quidem.* Sed omnis at commodi vel quasi. Fuga et **consequatur** molestias dicta vel provident et aspernatur. Dolorem molestias ipsa aut ~~facilis quae dolorem~~ eveniet dicta.`

	for i := 0; i < b.N; i++ {
		ParseMarkdown(in)
	}
}
