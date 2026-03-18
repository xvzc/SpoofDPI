package netutil

import (
	"net"
	"sync"
	"time"

	"github.com/xvzc/SpoofDPI/internal/cache"
)

// SessionCache manages UDP connections with LRU eviction policy and idle timeout.
type SessionCache[K comparable] struct {
	storage  cache.Cache[K]
	timeout  time.Duration
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewSessionCache creates a new pool with the specified capacity and timeout.
// Starts a background goroutine for expired connection cleanup.
func NewSessionCache[K comparable](
	capacity int,
	timeout time.Duration,
) *SessionCache[K] {
	p := &SessionCache[K]{
		timeout: timeout,
		stopCh:  make(chan struct{}),
	}

	onInvalidate := func(k K, v any) {
		if conn, ok := v.(*IdleTimeoutConn); ok {
			_ = conn.Conn.Close()
		}
	}

	p.storage = cache.NewLRUCache(capacity, onInvalidate)

	// Cleanup interval: half of timeout, min 10s, max 60s
	cleanupInterval := timeout / 2
	cleanupInterval = max(cleanupInterval, 10*time.Second)
	cleanupInterval = min(cleanupInterval, 60*time.Second)

	go p.cleanupLoop(cleanupInterval)
	return p
}

// Store adds a connection to the cache and returns the wrapped connection.
// If the key already exists, the old connection is closed and evicted first.
// If capacity is full, evicts the least recently used connection.
func (p *SessionCache[K]) Store(key K, rawConn net.Conn) *IdleTimeoutConn {
	wrapper := NewIdleTimeoutConn(rawConn, p.timeout)
	wrapper.Key = key

	wrapper.onActivity = func() {
		p.storage.Fetch(key)
	}

	wrapper.onClose = func() {
		p.Evict(key)
	}

	p.storage.Store(key, wrapper, nil)

	return wrapper
}

// Fetch retrieves a connection from the pool, refreshing its LRU status.
func (p *SessionCache[K]) Fetch(key K) (*IdleTimeoutConn, bool) {
	if val, ok := p.storage.Fetch(key); ok {
		return val.(*IdleTimeoutConn), true
	}
	return nil, false
}

// Evict closes and removes the connection from the pool.
func (p *SessionCache[K]) Evict(key K) {
	p.storage.Evict(key)
}

// Has checks if the connection exists in the cache.
func (p *SessionCache[K]) Has(key K) bool {
	return p.storage.Has(key)
}

// Size returns the number of connections in the pool.
func (p *SessionCache[K]) Size() int {
	return p.storage.Size()
}

// Stop stops the background cleanup goroutine.
func (p *SessionCache[K]) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
}

// CloseAll closes all connections in the pool.
func (p *SessionCache[K]) CloseAll() {
	var toRemove []K
	_ = p.storage.ForEach(func(key K, value any) error {
		toRemove = append(toRemove, key)
		return nil
	})
	for _, k := range toRemove {
		p.Evict(k) // safely removes without deadlocking
	}
}

func (p *SessionCache[K]) cleanupLoop(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopCh:
			return
		case <-ticker.C:
			p.evictExpired()
		}
	}
}

func (p *SessionCache[K]) evictExpired() {
	now := time.Now()
	var toRemove []K
	_ = p.storage.ForEach(func(key K, value any) error {
		if conn, ok := value.(*IdleTimeoutConn); ok {
			if conn.IsExpired(now) {
				toRemove = append(toRemove, key)
			}
		}
		return nil
	})
	for _, k := range toRemove {
		p.Evict(k)
	}
}
