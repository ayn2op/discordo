// Used by MessageInput.searchMember to not overflow the gateway with redundent
// search requests.
package cache

import (
	"sync"
)

type Cache struct {
	items sync.Map
}

func NewCache() *Cache {
	return &Cache{items: sync.Map{}}
}

func (c *Cache) Create(query string, value uint) {
	c.items.Store(query, value)
}

func (c *Cache) Exists(query string) (ok bool) {
	_, ok = c.items.Load(query)
	return
}

func (c *Cache) Get(query string) uint {
	i, _ := c.items.Load(query)
	return i.(uint)
}

// Invalidate is only needed when a member leaves and the search query reaches
// the search limit.
// "aa", "ab", "ac", ..., "ay" // where length is longer than the limit
// if "ay" leaves, then "az" would not be loaded becaue it would not be
// returned by the search results because of the search limit
func (c *Cache) Invalidate(name string, limit uint) {
	for name != "" {
		if c.Exists(name) && c.Get(name) >= limit {
			for name != "" {
				c.items.Delete(name)
				name = name[:len(name)-1]
			}
		}
		name = name[:len(name)-1]
	}
}
