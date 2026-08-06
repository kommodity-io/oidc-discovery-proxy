package handler

import (
	"time"

	"github.com/hashicorp/golang-lru/v2/expirable"
)

// cacheSize of 0 makes the LRU unbounded; the proxy only ever caches a handful of well-known paths.
const cacheSize = 0

// responseCache is a thread-safe, per-instance TTL cache for proxied responses.
// Only successful (HTTP 200) responses are ever cached, so no status code needs to be stored.
type responseCache struct {
	cache *expirable.LRU[string, []byte]
}

// newResponseCache builds a cache with the given TTL. A TTL <= 0 disables caching entirely.
func newResponseCache(ttl time.Duration) *responseCache {
	if ttl <= 0 {
		return &responseCache{}
	}

	return &responseCache{
		cache: expirable.NewLRU[string, []byte](cacheSize, nil, ttl),
	}
}

func (c *responseCache) get(key string) ([]byte, bool) {
	if c.cache == nil {
		return nil, false
	}

	return c.cache.Get(key)
}

func (c *responseCache) set(key string, data []byte) {
	if c.cache == nil {
		return
	}

	c.cache.Add(key, data)
}
