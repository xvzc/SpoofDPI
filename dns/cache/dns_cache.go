package cache

import (
	"sync"
	"time"

	"github.com/xvzc/SpoofDPI/util"
)

type DNSCacheEntry struct {
	ip     string
	expiry time.Time
}

type DNSCache struct {
	cacheMap        map[string]DNSCacheEntry
	cacheLock       sync.RWMutex
	scrubbingTicker *time.Ticker
}

var dnsCache *DNSCache
var once sync.Once

func scrubDNSCache() {
	for {
		<-dnsCache.scrubbingTicker.C
		dnsCache.cacheLock.Lock()
		for k, v := range dnsCache.cacheMap {
			if v.expiry.Before(time.Now()) {
				delete(dnsCache.cacheMap, k)
			}
		}
		dnsCache.cacheLock.Unlock()
	}
}

func GetCache() *DNSCache {
	once.Do(func() {
		dnsCache = &DNSCache{
			cacheMap:  make(map[string]DNSCacheEntry),
			cacheLock: sync.RWMutex{},

			scrubbingTicker: time.NewTicker(time.Duration(1 * time.Hour)), // Hourly cache scrub
		}

		go scrubDNSCache()
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
		ip:     value,
		expiry: time.Now().Add(time.Duration(*&util.GetConfig().DnsCacheTTL) * time.Second),
	}
}

func (d *DNSCache) Get(key string) (string, bool) {
	d.cacheLock.RLock() // Read lock

	if value, ok := d.cacheMap[key]; ok {
		if value.expiry.Before(time.Now()) {
			ip := value.ip

			d.cacheLock.RUnlock() // Read unlock
			return ip, true
		} else {
			d.cacheLock.RUnlock() // Read unlock

			d.cacheLock.Lock() // Lock for writing
			delete(d.cacheMap, key)
			d.cacheLock.Unlock() // Unlock

			return "", false
		}
	}

	d.cacheLock.RUnlock() // Read unlock
	return "", false
}
