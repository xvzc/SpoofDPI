package datastruct

import (
	"fmt"
	"hash/fnv" // FNV-1a: A fast, non-cryptographic hash function
	"sync"
	"time"
)

// ttlCacheItem represents a single cached item using generics.
// T is the generic type of the value being stored.
type ttlCacheItem[T any] struct {
	value     T         // The cached data of type T.
	expiresAt time.Time // The time when the item expires.
}

// isExpired checks if the item has expired.
func (i ttlCacheItem[T]) isExpired() bool {
	if i.expiresAt.IsZero() {
		// Zero time means no expiration.
		return false
	}
	return time.Now().After(i.expiresAt)
}

// ttlCacheShard represents a single, thread-safe shard of the cache.
type ttlCacheShard[T any] struct {
	items map[string]ttlCacheItem[T] // items holds the cache data for this shard.
	mu    sync.RWMutex
}

// TTLCache is a high-performance, sharded, generic TTL cache.
type TTLCache[T any] struct {
	shards []*ttlCacheShard[T] // A slice of cache shards.
}

// NewTTLCache creates a new sharded TTL cache with a background janitor goroutine.
// numShards specifies the number of shards to create and must be greater than 0.
// cleanupInterval specifies how often the janitor should run.
func NewTTLCache[T any](numShards uint64, cleanupInterval time.Duration) *TTLCache[T] {
	if numShards == 0 {
		panic(fmt.Errorf("ttlcache: numShards must be greater than 0, got %d", numShards))
	}

	c := &TTLCache[T]{
		shards: make([]*ttlCacheShard[T], numShards),
	}

	for i := uint64(0); i < numShards; i++ {
		c.shards[i] = &ttlCacheShard[T]{
			items: make(map[string]ttlCacheItem[T]),
		}
	}

	// Start the background janitor goroutine.
	// This goroutine is self-managing and terminates when the program exits.
	go c.janitor(cleanupInterval)

	return c
}

// getShard maps a key to its corresponding cache shard using a hash function.
func (c *TTLCache[T]) getShard(key string) *ttlCacheShard[T] {
	hasher := fnv.New64a()
	hasher.Write([]byte(key))
	hash := hasher.Sum64()
	return c.shards[hash%uint64(len(c.shards))]
}

// --- Public API ---

// Set adds an item to the cache, replacing any existing item.
// If ttl is 0 or negative, the item will never expire (passive-only).
func (c *TTLCache[T]) Set(key string, value T, ttl time.Duration) {
	var expiresAt time.Time
	if ttl > 0 {
		expiresAt = time.Now().Add(ttl)
	}
	newItem := ttlCacheItem[T]{
		value:     value,
		expiresAt: expiresAt,
	}
	shard := c.getShard(key)
	shard.mu.Lock()
	shard.items[key] = newItem
	shard.mu.Unlock()
}

// Get retrieves an item from the cache.
// It returns the item (of type T) and true if found and not expired.
// Otherwise, it returns the zero value of T and false.
func (c *TTLCache[T]) Get(key string) (T, bool) {
	shard := c.getShard(key)
	shard.mu.RLock()
	i, ok := shard.items[key]
	shard.mu.RUnlock()

	if !ok {
		var zero T
		return zero, false // 1. Cache miss.
	}

	// 2. Passive expiration: item found but is expired.
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
		var zero T
		return zero, false // Return as expired.
	}

	// 3. Cache hit.
	return i.value, true
}

// Delete removes an item from the cache.
func (c *TTLCache[T]) Delete(key string) {
	shard := c.getShard(key)
	shard.mu.Lock()
	delete(shard.items, key)
	shard.mu.Unlock()
}

// --- Background Janitor ---

// ForceCleanup actively scans all shards and deletes expired items.
// This is called periodically by the janitor but can also be called manually.
func (c *TTLCache[T]) ForceCleanup() {
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

// janitor runs the cleanup goroutine, calling ForceCleanup at the specified interval.
func (c *TTLCache[T]) janitor(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		c.ForceCleanup()
	}
}
