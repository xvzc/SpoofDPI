package cache

import "time"

// options holds all possible settings for a Set operation.
// Both caches will use this, but will only read what they need.
type options struct {
	ttl        time.Duration
	insertOnly bool
	override   bool
	// We could add other options here later, e.g.:
	// cost int
}

func Options() *options {
	return &options{}
}

func (o *options) WithTTL(ttl time.Duration) *options {
	o.ttl = ttl
	return o
}

func (o *options) WithOverride(override bool) *options {
	o.override = override
	return o
}

func (o *options) InsertOnly(insertOnly bool) *options {
	o.insertOnly = insertOnly
	return o
}

// Cache is the unified interface for all cache implementations.
// The Set method accepts a variadic list of options.
type Cache interface {
	// Get retrieves a value from the cache.
	Get(key string) (any, bool)
	// Set adds a value to the cache, applying any provided options.
	Set(key string, value any, opts *options)
}
