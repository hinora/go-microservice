package goservice

import (
	"encoding/json"
	"sync"
	"time"
)

// CacheConfig configures result caching for an action.
type CacheConfig struct {
	// TTL is how long a cached result is kept. 0 means entries never expire.
	TTL time.Duration
	// Keys restricts which parameter keys are included in the cache key.
	// An empty slice means all parameters are used.
	Keys []string
}

// cacheEntry holds a cached value and its expiry time.
type cacheEntry struct {
	value     interface{}
	expiresAt time.Time // zero value means never expires
}

// memoryCache is a simple TTL-aware in-memory cache safe for concurrent use.
type memoryCache struct {
	mu      sync.RWMutex
	entries map[string]*cacheEntry
}

func newMemoryCache() *memoryCache {
	c := &memoryCache{entries: make(map[string]*cacheEntry)}
	go c.gc()
	return c
}

// get returns the cached value for key. The second return is false on a miss or expiry.
func (c *memoryCache) get(key string) (interface{}, bool) {
	c.mu.RLock()
	e, ok := c.entries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
		return nil, false
	}
	return e.value, true
}

// set stores value under key with the given TTL (0 = never expire).
func (c *memoryCache) set(key string, value interface{}, ttl time.Duration) {
	entry := &cacheEntry{value: value}
	if ttl > 0 {
		entry.expiresAt = time.Now().Add(ttl)
	}
	c.mu.Lock()
	c.entries[key] = entry
	c.mu.Unlock()
}

// del removes entries matching pattern. A trailing "*" is treated as a prefix wildcard.
func (c *memoryCache) del(pattern string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(pattern) > 0 && pattern[len(pattern)-1] == '*' {
		prefix := pattern[:len(pattern)-1]
		for k := range c.entries {
			if len(k) >= len(prefix) && k[:len(prefix)] == prefix {
				delete(c.entries, k)
			}
		}
	} else {
		delete(c.entries, pattern)
	}
}

// gc periodically removes expired entries.
func (c *memoryCache) gc() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		c.mu.Lock()
		for k, e := range c.entries {
			if !e.expiresAt.IsZero() && now.After(e.expiresAt) {
				delete(c.entries, k)
			}
		}
		c.mu.Unlock()
	}
}

// buildCacheKey creates a deterministic string key from the action name and params.
// When cfg.Keys is non-empty only those param keys are included in the key.
func buildCacheKey(actionName string, params interface{}, cfg CacheConfig) string {
	var p interface{} = params
	if len(cfg.Keys) > 0 {
		if m, ok := params.(map[string]interface{}); ok {
			filtered := make(map[string]interface{}, len(cfg.Keys))
			for _, k := range cfg.Keys {
				filtered[k] = m[k]
			}
			p = filtered
		}
	}
	b, _ := json.Marshal(p)
	return actionName + ":" + string(b)
}
