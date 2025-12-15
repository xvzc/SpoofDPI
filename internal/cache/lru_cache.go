package cache

import (
	"container/list"
	"sync"
)

var _ Cache = (*LRUCache)(nil)

// lruEntry represents the value stored in the cache and the linked list node.
type lruEntry struct {
	key   string
	value any
	// expiry time.Time field removed
}

// LRUCache is a concurrent, fixed-size cache with an LRU eviction policy.
type LRUCache struct {
	capacity int
	mu       sync.RWMutex

	// list is a doubly linked list used for tracking access order.
	// Front is Most Recently Used (MRU), Back is Least Recently Used (LRU).
	list *list.List

	// cache maps the key to the list element (*list.Element) which holds the lruEntry.
	cache map[string]*list.Element
}

// NewLRUCache creates a new LRU Cache instance with the given capacity.
// Capacity must be greater than zero.
func NewLRUCache(capacity int) Cache {
	if capacity <= 0 {
		// Default to a sensible minimum capacity if input is invalid
		capacity = 100
	}
	return &LRUCache{
		capacity: capacity,
		list:     list.New(),
		cache:    make(map[string]*list.Element, capacity),
	}
}

// isExpired function removed

// evictOldest removes the least recently used item from the cache.
func (c *LRUCache) evictOldest() {
	// Element is at the back of the list (LRU)
	tail := c.list.Back()
	if tail != nil {
		c.removeElement(tail)
	}
}

// removeElement removes a specific list element from both the list and the map.
func (c *LRUCache) removeElement(e *list.Element) {
	c.list.Remove(e)
	entry := e.Value.(*lruEntry)
	delete(c.cache, entry.key)
}

// Get retrieves a value from the cache.
// If found, the item is promoted to Most Recently Used (MRU).
func (c *LRUCache) Get(key string) (any, bool) {
	// Use RLock for concurrent safe read operations
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check if key exists in the map
	if element, ok := c.cache[key]; ok {
		// No TTL check needed.

		// Promote to Most Recently Used (MRU)
		c.list.MoveToFront(element)

		entry := element.Value.(*lruEntry)
		return entry.value, true
	}

	return nil, false
}

// Set adds a value to the cache, applying any provided options.
func (c *LRUCache) Set(key string, value any, opts *options) bool {
	// Use Write Lock for modification operations
	c.mu.Lock()
	defer c.mu.Unlock()

	if opts == nil {
		opts = Options()
	}

	element, ok := c.cache[key]
	if ok && opts.skipExisting {
		return false
	}

	if !ok && opts.updateExistingOnly {
		return false
	}

	if ok {
		entry := element.Value.(*lruEntry)
		entry.value = value

		c.list.MoveToFront(element)
		return true
	}

	// Key is new: Create a new entry
	entry := &lruEntry{
		key:   key,
		value: value,
	}

	// Add new entry to the front of the list (MRU) and to the map
	element = c.list.PushFront(entry)
	c.cache[key] = element

	// Check for eviction if capacity is exceeded
	if c.list.Len() > c.capacity {
		c.evictOldest()
	}

	return true
}
