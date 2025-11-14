package cache

import "time"

// options holds all possible settings for a Set operation.
// Both caches will use this, but will only read what they need.
type options struct {
	ttl time.Duration
	// We could add other options here later, e.g.:
	// cost int
}

// SetOption is a function type that applies a setting to an options struct.
// This is the core of the "functional options" pattern.
type SetOption func(*options)

// applyOpts creates an options struct, applies all SetOption functions,
// and returns the result.
func applyOpts(opts ...SetOption) options {
	// Load default options
	opt := options{
		ttl: 0, // 0 means no expiry (or a default expiry)
	}

	// Apply all provided options
	for _, applyOpt := range opts {
		applyOpt(&opt)
	}
	return opt
}

// WithTTL returns a SetOption function.
// When called, this function sets the TTL value in the options.
func WithTTL(ttl time.Duration) SetOption {
	return func(o *options) {
		o.ttl = ttl
	}
}

// Cache is the unified interface for all cache implementations.
// The Set method accepts a variadic list of options.
type Cache interface {
	// Get retrieves a value from the cache.
	Get(key string) (any, bool)
	// Set adds a value to the cache, applying any provided options.
	Set(key string, value any, opts ...SetOption)
}
