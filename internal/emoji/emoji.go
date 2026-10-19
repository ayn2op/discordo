package emoji

import (
	"slices"
	"strings"
	"sync"

	"github.com/ayn2op/arikawa/v3/discord"
	"github.com/sahilm/fuzzy"
)

type Item struct {
	ID         discord.EmojiID
	Name       string
	InsertText string
	Display    string
	Custom     bool
}

// APIEmoji returns the formatted API emoji identifier for reactions.
func (i Item) APIEmoji() discord.APIEmoji {
	if i.Custom && i.ID.IsValid() {
		return discord.NewAPIEmoji(i.ID, i.Name)
	}
	return discord.APIEmoji(i.InsertText)
}

type itemList []Item

func (l itemList) String(i int) string {
	return l[i].Name
}

func (l itemList) Len() int {
	return len(l)
}

var unicodeList = sync.OnceValue(func() []Item {
	items := make([]Item, 0, len(Unicode))
	for name, glyph := range Unicode {
		items = append(items, Item{
			Name:       name,
			InsertText: glyph,
			Display:    ":" + name + ": " + glyph,
			Custom:     false,
		})
	}
	slices.SortFunc(items, func(a, b Item) int {
		if a.Name < b.Name {
			return -1
		} else if a.Name > b.Name {
			return 1
		}
		return 0
	})
	return items
})

// Search searches custom emojis and standard Unicode emojis matching the query.
func Search(query string, custom []discord.Emoji, limit int) []Item {
	if limit <= 0 {
		limit = 20
	}

	customItems := make([]Item, 0, len(custom))
	for _, e := range custom {
		customItems = append(customItems, Item{
			ID:         e.ID,
			Name:       e.Name,
			InsertText: e.String(),
			Display:    ":" + e.Name + ":",
			Custom:     true,
		})
	}

	uItems := unicodeList()

	if query == "" {
		results := make([]Item, 0, limit)
		results = append(results, customItems...)
		if len(results) < limit {
			needed := limit - len(results)
			if needed > len(uItems) {
				needed = len(uItems)
			}
			results = append(results, uItems[:needed]...)
		}
		if len(results) > limit {
			return results[:limit]
		}
		return results
	}

	// Exact prefix matches on custom emojis get highest priority
	var results []Item
	lowerQuery := strings.ToLower(query)

	for _, item := range customItems {
		if strings.HasPrefix(strings.ToLower(item.Name), lowerQuery) {
			results = append(results, item)
			if len(results) >= limit {
				return results
			}
		}
	}

	// Fuzzy search across remaining custom emojis
	if len(customItems) > 0 {
		customMatches := fuzzy.FindFrom(query, itemList(customItems))
		for _, m := range customMatches {
			item := customItems[m.Index]
			if !slices.ContainsFunc(results, func(it Item) bool { return it.Name == item.Name && it.Custom }) {
				results = append(results, item)
				if len(results) >= limit {
					return results
				}
			}
		}
	}

	// Exact prefix matches on unicode emojis
	for _, item := range uItems {
		if strings.HasPrefix(strings.ToLower(item.Name), lowerQuery) {
			results = append(results, item)
			if len(results) >= limit {
				return results
			}
		}
	}

	// Fuzzy search across unicode emojis
	uMatches := fuzzy.FindFrom(query, itemList(uItems))
	for _, m := range uMatches {
		item := uItems[m.Index]
		if !slices.ContainsFunc(results, func(it Item) bool { return it.Name == item.Name && !it.Custom }) {
			results = append(results, item)
			if len(results) >= limit {
				return results
			}
		}
	}

	return results
}
