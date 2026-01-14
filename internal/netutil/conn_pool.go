package netutil

import (
	"container/list"
	"net"
	"sync"
	"time"
)

// ConnPool manages UDP connections with LRU eviction policy and idle timeout.
type ConnPool struct {
	capacity int
	timeout  time.Duration
	cache    map[string]*list.Element
	ll       *list.List
	mu       sync.Mutex
	stopCh   chan struct{}
	stopOnce sync.Once
}

// PooledConn wraps net.Conn with LRU tracking and deadline management.
type PooledConn struct {
	net.Conn
	pool      *ConnPool
	key       string
	timeout   time.Duration
	expiredAt time.Time
}

type connEntry struct {
	key  string
	conn *PooledConn
}

// NewConnPool creates a new pool with the specified capacity and timeout.
// Starts a background goroutine for expired connection cleanup.
func NewConnPool(capacity int, timeout time.Duration) *ConnPool {
	p := &ConnPool{
		capacity: capacity,
		timeout:  timeout,
		cache:    make(map[string]*list.Element),
		ll:       list.New(),
		stopCh:   make(chan struct{}),
	}

	// Cleanup interval: half of timeout, min 10s, max 60s
	cleanupInterval := timeout / 2
	if cleanupInterval < 10*time.Second {
		cleanupInterval = 10 * time.Second
	}
	if cleanupInterval > 60*time.Second {
		cleanupInterval = 60 * time.Second
	}

	go p.cleanupLoop(cleanupInterval)
	return p
}

// Add adds a connection to the pool and returns the wrapped connection.
// If capacity is full, evicts the least recently used connection.
func (p *ConnPool) Add(key string, rawConn net.Conn) *PooledConn {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Evict if capacity is reached
	if p.ll.Len() >= p.capacity {
		p.evictOldest()
	}

	now := time.Now()
	expiredAt := now.Add(p.timeout)

	wrapper := &PooledConn{
		Conn:      rawConn,
		pool:      p,
		key:       key,
		timeout:   p.timeout,
		expiredAt: expiredAt,
	}

	_ = rawConn.SetDeadline(expiredAt)

	elem := p.ll.PushFront(&connEntry{key: key, conn: wrapper})
	p.cache[key] = elem

	return wrapper
}

// Remove closes and removes the connection from the pool.
func (p *ConnPool) Remove(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if elem, ok := p.cache[key]; ok {
		p.removeElement(elem)
	}
}

// Size returns the number of connections in the pool.
func (p *ConnPool) Size() int {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.ll.Len()
}

// Stop stops the background cleanup goroutine.
func (p *ConnPool) Stop() {
	p.stopOnce.Do(func() {
		close(p.stopCh)
	})
}

// CloseAll closes all connections in the pool.
func (p *ConnPool) CloseAll() {
	p.mu.Lock()
	defer p.mu.Unlock()

	elem := p.ll.Front()
	for elem != nil {
		next := elem.Next()
		p.removeElement(elem)
		elem = next
	}
}

func (p *ConnPool) cleanupLoop(interval time.Duration) {
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

func (p *ConnPool) evictExpired() {
	p.mu.Lock()
	defer p.mu.Unlock()

	now := time.Now()
	elem := p.ll.Back()
	for elem != nil {
		// Save next before potential removal
		next := elem.Prev()
		e := elem.Value.(*connEntry)
		if now.After(e.conn.expiredAt) {
			p.removeElement(elem)
		}
		elem = next
	}
}

func (p *ConnPool) evictOldest() {
	if elem := p.ll.Back(); elem != nil {
		p.removeElement(elem)
	}
}

func (p *ConnPool) removeElement(elem *list.Element) {
	e := elem.Value.(*connEntry)
	_ = e.conn.Conn.Close()
	p.ll.Remove(elem)
	delete(p.cache, e.key)
}

func (p *ConnPool) touch(key string) {
	p.mu.Lock()
	defer p.mu.Unlock()
	if elem, ok := p.cache[key]; ok {
		p.ll.MoveToFront(elem)
	}
}

func (c *PooledConn) refreshDeadline() {
	c.expiredAt = time.Now().Add(c.timeout)
	_ = c.SetDeadline(c.expiredAt)
	c.pool.touch(c.key)
}

// Read reads data and refreshes the deadline on success.
func (c *PooledConn) Read(b []byte) (n int, err error) {
	n, err = c.Conn.Read(b)
	if n > 0 {
		c.refreshDeadline()
	}
	return
}

// Write writes data and refreshes the deadline on success.
func (c *PooledConn) Write(b []byte) (n int, err error) {
	n, err = c.Conn.Write(b)
	if n > 0 {
		c.refreshDeadline()
	}
	return
}

// Close removes the connection from the pool (underlying close handled by pool).
func (c *PooledConn) Close() error {
	c.pool.Remove(c.key)
	return nil
}
