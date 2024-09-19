package markdown

import (
	"regexp"

	"github.com/ayn2op/discordo/internal/config"
)

var (
	boldRe          = regexp.MustCompile(`(?ms)\*\*(.*?)\*\*`)
	italicRe        = regexp.MustCompile(`(?ms)\*(.*?)\*`)
	underlineRe     = regexp.MustCompile(`(?ms)__(.*?)__`)
	strikethroughRe = regexp.MustCompile(`(?ms)~~(.*?)~~`)
	codeblockRe     = regexp.MustCompile("(?ms)`" + `([^` + "`" + `\n]+)` + "`")
	emoteRe         = regexp.MustCompile(`<(:[a-zA-Z0-9]+:)[0-9]+>`)

	cfg *config.Config
	emoteColor string
)

func Init() {
	var err error
	cfg, err := config.Load()
	if err != nil {
		return
	}
	
	emoteColor = cfg.Theme.MessagesText.EmoteColor
}

func Parse(input string) string {
	input = boldRe.ReplaceAllString(input, "[::b]$1[::B]")
	input = italicRe.ReplaceAllString(input, "[::i]$1[::I]")
	input = underlineRe.ReplaceAllString(input, "[::u]$1[::U]")
	input = strikethroughRe.ReplaceAllString(input, "[::s]$1[::S]")
	input = codeblockRe.ReplaceAllString(input, "[::r]$1[::R]")
	input = emoteRe.ReplaceAllString(input, "[" + emoteColor + "]$1[-:-:-]")

	return input
}
