package discordmd

import (
	"regexp"
)

var (
	boldRegex          = regexp.MustCompile(`(?ms)\*\*(.*?)\*\*`)
	italicRegex        = regexp.MustCompile(`(?ms)\*(.*?)\*`)
	underlineRegex     = regexp.MustCompile(`(?ms)__(.*?)__`)
	strikeThroughRegex = regexp.MustCompile(`(?ms)~~(.*?)~~`)
	spoilerRegex       = regexp.MustCompile(`(?ms)\|\|(.*?)\|\|`)
)

// Parse parses Discord-flavored markdown to tview's [Color Tags].
//
// [Color Tags]: https://pkg.go.dev/github.com/rivo/tview#hdr-Colors
func Parse(md string) string {
	md = boldRegex.ReplaceAllString(md, "[::b]$1[::-]")
	md = italicRegex.ReplaceAllString(md, "[::i]$1[::-]")
	md = underlineRegex.ReplaceAllString(md, "[::u]$1[::-]")
	md = strikeThroughRegex.ReplaceAllString(md, "[::s]$1[::-]")

	md = spoilerRegex.ReplaceAllStringFunc(md, replaceWithNothing)

	return md
}

func replaceWithNothing(s string) string {

	runes := []rune(s)

	for i := range runes {
		runes[i] = ' '
	}

	return "[#383838:#FFFFFF:]" + string(runes) + "[-:-:]"
}

	