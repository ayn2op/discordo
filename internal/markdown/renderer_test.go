package markdown

import (
	"strings"
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/ayn2op/ningen/v3/discordmd"
	"github.com/gdamore/tcell/v3"
	"github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
)

func TestRendererMasksSpoilers(t *testing.T) {
	source := []byte("before ||secret|| after")
	node := parser.NewParser(
		parser.WithBlockParsers(discordmd.BlockParsers()...),
		parser.WithInlineParsers(discordmd.InlineParserWithLink()...),
	).Parse(text.NewReader(source))

	for _, test := range []struct {
		mask bool
		want string
	}{
		{true, "before [spoiler] after"},
		{false, "before secret after"},
	} {
		rendered := NewRenderer(&config.Config{Markdown: config.MarkdownConfig{MaskSpoilers: test.mask}}).RenderText(source, node, tcell.StyleDefault)
		var got strings.Builder
		for _, line := range rendered {
			for _, segment := range line {
				got.WriteString(segment.Text)
			}
		}
		if got.String() != test.want {
			t.Fatalf("mask=%t: got %q, want %q", test.mask, got.String(), test.want)
		}
	}
}
