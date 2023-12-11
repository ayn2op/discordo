package markdown

import (
	"regexp"
)

var (
	boldRe          = regexp.MustCompile(`(?ms)\*\*(.*?)\*\*`)
	italicRe        = regexp.MustCompile(`(?ms)\*(.*?)\*`)
	underlineRe     = regexp.MustCompile(`(?ms)__(.*?)__`)
	strikethroughRe = regexp.MustCompile(`(?ms)~~(.*?)~~`)
	codeblockRe     = regexp.MustCompile("(?ms)`" + `([^` + "`" + `\n]+)` + "`")
)

func Parse(input string) string {
	input = boldRe.ReplaceAllString(input, "[::b]$1[::-]")
	input = italicRe.ReplaceAllString(input, "[::i]$1[::-]")
	input = underlineRe.ReplaceAllString(input, "[::u]$1[::-]")
	input = strikethroughRe.ReplaceAllString(input, "[::s]$1[::-]")
	input = codeblockRe.ReplaceAllString(input, "[::r]$1[::-]")
	return input
}
