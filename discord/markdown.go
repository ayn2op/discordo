package discord

import "regexp"

var (
	boldRegex          = regexp.MustCompile(`(?m)\*\*(.*?)\*\*`)
	italicRegex        = regexp.MustCompile(`(?m)\*(.*?)\*`)
	underlineRegex     = regexp.MustCompile(`(?m)__(.*?)__`)
	strikeThroughRegex = regexp.MustCompile(`(?m)~~(.*?)~~`)
)

func ParseMarkdown(md string) string {
	var res string
	res = boldRegex.ReplaceAllString(md, "[::b]$1[::-]")
	res = italicRegex.ReplaceAllString(res, "[::i]$1[::-]")
	res = underlineRegex.ReplaceAllString(res, "[::u]$1[::-]")
	res = strikeThroughRegex.ReplaceAllString(res, "[::s]$1[::-]")

	return res
}
