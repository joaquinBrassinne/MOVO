package nominatim

import (
	"sync"
	"time"

	"github.com/joaquinBrassinne/MOVO/internal/business"
)

type cached struct {
	place business.Place
	err   error
}

type cacheEntry struct {
	cached
	expires time.Time
}

// cache is a small in-memory cache with a time to live and a size cap. When it is
// full the oldest entry goes first.
type cache struct {
	mu       sync.Mutex
	entries  map[string]cacheEntry
	order    []string // keys, oldest first
	capacity int
	ttl      time.Duration
}

func newCache(capacity int, ttl time.Duration) *cache {
	return &cache{entries: make(map[string]cacheEntry, capacity), capacity: capacity, ttl: ttl}
}

func (c *cache) get(key string) (cached, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expires) {
		return cached{}, false
	}
	return entry.cached, true
}

func (c *cache) put(key string, value cached) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; !exists {
		if len(c.order) >= c.capacity {
			delete(c.entries, c.order[0])
			c.order = c.order[1:]
		}
		c.order = append(c.order, key)
	}
	c.entries[key] = cacheEntry{cached: value, expires: time.Now().Add(c.ttl)}
}
