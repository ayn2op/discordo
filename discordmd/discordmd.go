package discordmd

import (
	"regexp"
)

var (
	boldRegex            = regexp.MustCompile(`(?ms)\*\*(.*?)\*\*`)
	italicRegex          = regexp.MustCompile(`(?ms)\*(.*?)\*`)
	underlineRegex       = regexp.MustCompile(`(?ms)__(.*?)__`)
	strikeThroughRegex   = regexp.MustCompile(`(?ms)~~(.*?)~~`)
	inlineCodeBlockRegex = regexp.MustCompile("(?ms)`" + `([^` + "`" + `\n]+)` + "`")
)

func Parse(input string) string {
	input = boldRegex.ReplaceAllString(input, "[::b]$1[::-]")
	input = italicRegex.ReplaceAllString(input, "[::i]$1[::-]")
	input = underlineRegex.ReplaceAllString(input, "[::u]$1[::-]")
	input = strikeThroughRegex.ReplaceAllString(input, "[::s]$1[::-]")
	input = inlineCodeBlockRegex.ReplaceAllString(input, "[::r]$1[::-]")
	return input
}
