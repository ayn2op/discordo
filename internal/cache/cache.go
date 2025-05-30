// This is not a real cache system, it only remembers what is the time
// the item has been cerated at, the returned chunked member count,
// and at what time it has been accessed.
// Used by MessageInput.searchMember to not overflow the gateway with redundent
// search requests.
package cache

import (
	"sync"
	"time"
)

type Cache struct {
	items sync.Map
	close chan struct{}
}

type item struct {
	lastAccessed int64
	creationTime int64
	memberCount  uint
}

func NewCache() *Cache {
	cache := &Cache {
		items: sync.Map{},
		close: make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				now := time.Now().Unix()
				lastMinute := now + 60
				last5Minutes := now + 60*5
				cache.items.Range(func(key any, val any) bool {
					i := val.(item)
					if lastMinute > i.lastAccessed ||
					   last5Minutes > i.creationTime {
						cache.items.Delete(key.(string))
					}
					return true
				})
			case <-cache.close:
				return
			}
		}
	}()

	return cache
}

func (c *Cache) Create(key string, memberCount uint) {
	c.items.Store(key, item{
		lastAccessed: time.Now().Unix(),
		creationTime: time.Now().Unix(),
		memberCount:  memberCount,
	})
}

func (c *Cache) Exists(key string) (ok bool) {
	if i, ok := c.items.Load(key); ok {
		c.items.Store(key, item{
			lastAccessed: time.Now().Unix(),
			creationTime: i.(item).creationTime,
			memberCount:  i.(item).memberCount,
		})
	}
	return
}

func (c *Cache) GetMemberCount(key string) uint {
	i, _ := c.items.Load(key)
	return i.(item).memberCount
}

func (c *Cache) Close() {
	c.close <- struct{}{}
	c.items = sync.Map{}
}
