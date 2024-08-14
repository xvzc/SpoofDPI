package cache

import (
	"sync"
	"time"

	"github.com/xvzc/SpoofDPI/util"
)

type DNSCacheEntry struct {
	ip           string
	expiry_timer *time.Timer
}

type DNSCache struct {
	cacheMap  map[string]DNSCacheEntry
	cacheLock sync.RWMutex
}

var dnsCache *DNSCache
var once sync.Once

func GetCache() *DNSCache {
	once.Do(func() {
		dnsCache = &DNSCache{
			cacheMap:  make(map[string]DNSCacheEntry),
			cacheLock: sync.RWMutex{},
		}
	})
	return dnsCache
}

// interface function for the inner map basically.
func (d *DNSCache) Set(key string, value string) {
	d.cacheLock.Lock()
	defer d.cacheLock.Unlock()
	if _, ok := d.cacheMap[key]; ok {
		return
	}

	d.cacheMap[key] = DNSCacheEntry{
		ip: value,
		expiry_timer: time.AfterFunc(time.Duration(*&util.GetConfig().DnsCacheTTL)*time.Second, func() {
			d.cacheLock.Lock()
			defer d.cacheLock.Unlock()
			delete(d.cacheMap, key)
		}),
	}
}

func (d *DNSCache) Get(key string) (string, bool) {
	d.cacheLock.RLock()
	defer d.cacheLock.RUnlock()

	if value, ok := d.cacheMap[key]; ok {
		return value.ip, true
	}

	return "", false
}
