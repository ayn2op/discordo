//go:build !no_syntax_highlighting

package markdown

import (
	"strings"
	"testing"

	"github.com/ayn2op/discordo/internal/config"
	"github.com/gdamore/tcell/v3"
)

func TestRendererWriteCodeBlock(t *testing.T) {
	// A shebang is what the shell lexer recognises a script by.
	const script = "#!/bin/bash\necho hi"
	for _, test := range []struct {
		name   string
		detect bool
		want   string
	}{
		{"does not guess the language by default", false, codeBlockIndent + "code\n"},
		{"guesses the language when enabled", true, codeBlockIndent + "code: analyzed\n"},
	} {
		t.Run(test.name, func(t *testing.T) {
			renderer := NewRenderer(&config.Config{Markdown: config.MarkdownConfig{DetectCodeLanguage: test.detect}})
			var builder Builder
			renderer.writeCodeBlock(&builder, script, "", tcell.StyleDefault)
			var got strings.Builder
			for _, span := range builder.Finish() {
				got.WriteString(span.Text)
			}
			if !strings.HasPrefix(got.String(), test.want) {
				t.Fatalf("got %q, want it to start with %q", got.String(), test.want)
			}
		})
	}
}
