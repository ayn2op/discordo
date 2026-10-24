package messageslist

import (
	"net/url"
	"slices"
	"strings"

	"github.com/ayn2op/arikawa/v3/discord"
	md "github.com/ayn2op/arikawa/v3/markdown"
	"github.com/yuin/goldmark/v2/ast"
)

// contentURLs returns the link destinations in content. root is its parsed markdown, or nil to parse it here.
func contentURLs(content string, root ast.Node) []string {
	src := []byte(content)
	if root == nil {
		root = md.Parse(src)
	}
	return urlsFromAST(root, src)
}

// urlsFromAST collects link destinations from an already-parsed markdown AST.
// source must be the byte slice the node was parsed from (AutoLink resolves its URL against it).
func urlsFromAST(node ast.Node, source []byte) []string {
	var urls []string
	_ = ast.Walk(node, func(current ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		switch link := current.(type) {
		case *ast.AutoLink:
			urls = append(urls, string(link.Destination.Bytes(source)))
		case *ast.Link:
			urls = append(urls, string(link.Destination.Bytes(source)))
		}

		return ast.WalkContinue, nil
	})

	return urls
}

func extractEmbedURLs(embeds []discord.Embed) []string {
	urls := make([]string, 0, len(embeds)*3)
	for _, embed := range embeds {
		if embed.URL != "" {
			urls = append(urls, embed.URL)
		}
		if embed.Image != nil && embed.Image.URL != "" {
			urls = append(urls, embed.Image.URL)
		}
		if embed.Video != nil && embed.Video.URL != "" {
			urls = append(urls, embed.Video.URL)
		}
	}
	return urls
}

// The schemes of the links that are opened.
const (
	schemeHTTP  = "http"
	schemeHTTPS = "https"
)

// messageURLs returns the web links of message. root is its parsed content, or nil to parse it here.
func (m Model) messageURLs(message discord.Message, root ast.Node) []string {
	embeds := message.Embeds
	if !m.cfg.UI.MessagesList.Embeds.Enabled {
		embeds = nil
	}
	combined := slices.Concat(contentURLs(message.Content, root), extractEmbedURLs(embeds))

	urls := make([]string, 0, len(combined))
	seen := make(map[string]struct{}, len(combined))
	for _, link := range combined {
		link = strings.TrimSpace(link)
		// Only web links are opened, as one of another scheme could run a program.
		if parsed, err := url.Parse(link); err != nil || parsed.Scheme != schemeHTTP && parsed.Scheme != schemeHTTPS {
			continue
		}

		if _, ok := seen[link]; ok {
			continue
		}

		seen[link] = struct{}{}
		urls = append(urls, link)
	}
	return urls
}
