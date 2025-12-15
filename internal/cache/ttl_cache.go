package cache

import (
	"fmt"
	"hash/fnv" // FNV-1a: A fast, non-cryptographic hash function
	"sync"
	"time"
)

var _ Cache = (*TTLCache)(nil)

// ttlCacheItem represents a single cached item using generics.
type ttlCacheItem struct {
	value     any       // The cached data of type T.
	expiresAt time.Time // The time when the item expires.
}

// isExpired checks if the item has expired.
func (i ttlCacheItem) isExpired() bool {
	if i.expiresAt.IsZero() {
		// zero time means no expiration.
		return false
	}
	return time.Now().After(i.expiresAt)
}

// ttlCacheShard represents a single, thread-safe shard of the cache.
type ttlCacheShard struct {
	items map[string]ttlCacheItem // items holds the cache data for this shard.
	mu    sync.RWMutex
}

type TTLCacheAttrs struct {
	NumOfShards     uint8
	CleanupInterval time.Duration
}

// TTLCache is a high-performance, sharded, generic TTL cache.
type TTLCache struct {
	shards []*ttlCacheShard // A slice of cache shards.
}

// NewTTLCache creates a new sharded TTL cache with a background janitor goroutine.
// numShards specifies the number of shards to create and must be greater than 0.
// cleanupInterval specifies how often the janitor should run.
func NewTTLCache(
	attrs TTLCacheAttrs,
) *TTLCache {
	if attrs.NumOfShards == 0 {
		panic(
			fmt.Errorf("number of shards must be greater than 0, got %d", attrs.NumOfShards),
		)
	}

	c := &TTLCache{
		shards: make([]*ttlCacheShard, attrs.NumOfShards),
	}

	for i := range attrs.NumOfShards {
		c.shards[i] = &ttlCacheShard{
			items: make(map[string]ttlCacheItem),
		}
	}

	// Start the background janitor goroutine.
	// This goroutine is self-managing and terminates when the program exits.
	go c.janitor(attrs.CleanupInterval)

	return c
}

// janitor runs the cleanup goroutine, calling ForceCleanup at the specified interval.
func (c *TTLCache) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		c.ForceCleanup()
	}
}

// getShard maps a key to its corresponding cache shard using a hash function.
func (c *TTLCache) getShard(key string) *ttlCacheShard {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	hash := hasher.Sum64()
	return c.shards[hash%uint64(len(c.shards))]
}

// ┌─────────────┐
// │ PUBLIC APIs │
// └─────────────┘
// Set adds an item to the cache, replacing any existing item.
// If ttl is 0 or negative, the item will never expire (passive-only).
func (c *TTLCache) Set(key string, value any, opts *options) bool {
	shard := c.getShard(key)

	shard.mu.Lock()
	defer shard.mu.Unlock()

	if opts == nil {
		opts = Options()
	}

	if opts.ttl == 0 {
		return false
	}

	_, ok := shard.items[key]
	if ok && opts.skipExisting {
		return false
	}

	if !ok && opts.updateExistingOnly {
		return false
	}

	expiresAt := time.Now().Add(opts.ttl)
	newItem := ttlCacheItem{
		value:     value,
		expiresAt: expiresAt,
	}
	shard.items[key] = newItem

	return true
}

// Get retrieves an item from the cache.
// It returns the item (of type T) and true if found and not expired.
// Otherwise, it returns the zero value of T and false.
func (c *TTLCache) Get(key string) (any, bool) {
	shard := c.getShard(key)
	shard.mu.RLock()
	i, ok := shard.items[key]
	shard.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Passive expiration: item found but is expired.
	if i.isExpired() {
		// Item is expired, so acquire a write lock to delete it.
		shard.mu.Lock()
		// Double-check: ensure the item hasn't been replaced
		// by another goroutine while we were waiting for the write lock.
		if currentItem, ok := shard.items[key]; ok {
			if time.Time.Equal(currentItem.expiresAt, i.expiresAt) {
				delete(shard.items, key)
			}
		}

		shard.mu.Unlock()
		return nil, false // Return as expired.
	}

	// Cache hit.
	return i.value, true
}

// Delete removes an item from the cache.
func (c *TTLCache) Delete(key string) {
	shard := c.getShard(key)
	shard.mu.Lock()
	delete(shard.items, key)
	shard.mu.Unlock()
}

// ForceCleanup actively scans all shards and deletes expired items.
// This is called periodically by the janitor but can also be called manually.
func (c *TTLCache) ForceCleanup() {
	now := time.Now()
	for _, shard := range c.shards {
		shard.mu.Lock()
		for key, i := range shard.items {
			if !i.expiresAt.IsZero() && now.After(i.expiresAt) {
				delete(shard.items, key)
			}
		}
		shard.mu.Unlock()
	}
}
