package handler

import (
	"time"

	lru "github.com/hashicorp/golang-lru/v2/expirable"
)

// cacheSize of 0 makes the LRU unbounded; the proxy only ever caches a handful of well-known paths.
const cacheSize = 0

type cacheEntry struct {
	data       []byte
	statusCode int
}

// responseCache is a thread-safe, per-instance TTL cache for proxied responses.
type responseCache struct {
	cache *lru.LRU[string, cacheEntry]
}

// newResponseCache builds a cache with the given TTL. A TTL <= 0 disables caching entirely.
func newResponseCache(ttl time.Duration) *responseCache {
	if ttl <= 0 {
		return &responseCache{}
	}

	return &responseCache{
		cache: lru.NewLRU[string, cacheEntry](cacheSize, nil, ttl),
	}
}

func (c *responseCache) get(key string) ([]byte, int, bool) {
	if c.cache == nil {
		return nil, 0, false
	}

	entry, found := c.cache.Get(key)
	if !found {
		return nil, 0, false
	}

	return entry.data, entry.statusCode, true
}

func (c *responseCache) set(key string, data []byte, statusCode int) {
	if c.cache == nil {
		return
	}

	c.cache.Add(key, cacheEntry{data: data, statusCode: statusCode})
}
