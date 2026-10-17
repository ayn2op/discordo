package markdown

import (
	"strings"

	"github.com/ayn2op/tview"
	"github.com/ayn2op/tview/text"
)

// Builder builds the spans of a text.
type Builder struct {
	spans []text.Span
}

// Write appends value in style.
func (b *Builder) Write(value string, style tview.Style) {
	if value != "" {
		b.spans = append(b.spans, text.Span{Text: value, Style: style})
	}
}

// WriteSpans appends spans.
func (b *Builder) WriteSpans(spans ...text.Span) {
	b.spans = append(b.spans, spans...)
}

// LineEmpty reports whether nothing was written since the last line break.
func (b *Builder) LineEmpty() bool {
	return len(b.spans) == 0 || strings.HasSuffix(b.spans[len(b.spans)-1].Text, "\n")
}

// NewLine starts a new line.
func (b *Builder) NewLine() {
	b.Write("\n", tview.Style{})
}

// Finish returns the spans, without a line break at the end, which would add an empty line.
func (b *Builder) Finish() []text.Span {
	if n := len(b.spans); n > 0 {
		last := &b.spans[n-1]
		if last.Text = strings.TrimSuffix(last.Text, "\n"); last.Text == "" {
			b.spans = b.spans[:n-1]
		}
	}
	return b.spans
}
