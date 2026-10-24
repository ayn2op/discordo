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
// src must be the byte slice the node was parsed from (AutoLink resolves its URL against it).
func urlsFromAST(node ast.Node, src []byte) []string {
	var urls []string
	_ = ast.Walk(node, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering {
			switch n := n.(type) {
			case *ast.AutoLink:
				urls = append(urls, string(n.Destination.Bytes(src)))
			case *ast.Link:
				urls = append(urls, string(n.Destination.Bytes(src)))
			}
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

// messageURLs returns the web links of msg. root is its parsed content, or nil to parse it here.
func (m Model) messageURLs(msg discord.Message, root ast.Node) []string {
	embeds := msg.Embeds
	if !m.cfg.UI.MessagesList.Embeds.Enabled {
		embeds = nil
	}
	combined := slices.Concat(contentURLs(msg.Content, root), extractEmbedURLs(embeds))

	urls := make([]string, 0, len(combined))
	seen := make(map[string]struct{}, len(combined))
	for _, u := range combined {
		u = strings.TrimSpace(u)
		// Only web links are opened, as one of another scheme could run a program.
		if parsed, err := url.Parse(u); err != nil || parsed.Scheme != schemeHTTP && parsed.Scheme != schemeHTTPS {
			continue
		}
		if _, ok := seen[u]; ok {
			continue
		}
		seen[u] = struct{}{}
		urls = append(urls, u)
	}
	return urls
}
