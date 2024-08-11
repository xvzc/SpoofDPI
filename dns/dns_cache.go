package dns

import (
	"errors"
	"sync"
	"time"

	"github.com/xvzc/SpoofDPI/util"
)

type DNSCacheEntry struct {
	ip     string
	expire time.Time
}

func (d *DNSCacheEntry) Expired() bool {
	return time.Now().After(d.expire)
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

		go func() {
			for {
				time.Sleep(time.Duration(*util.GetConfig().DNSCacheExpiry) * time.Second)
				dnsCache.cacheLock.Lock()
				for key, value := range dnsCache.cacheMap {
					if value.Expired() {
						delete(dnsCache.cacheMap, key)
					}
				}
			}
		}()
	})
	return dnsCache
}

// interface function for the inner map basically.
func (d *DNSCache) Set(key string, value string) {
	d.cacheLock.Lock()

	d.cacheMap[key] = DNSCacheEntry{
		ip:     value,
		expire: time.Now().Add(time.Duration(*util.GetConfig().DNSCacheExpiry) * time.Second),
	}

	d.cacheLock.Unlock()
}

func (d *DNSCache) Get(key string) (string, error) {
	d.cacheLock.RLock()
	defer d.cacheLock.RUnlock()

	if value, ok := d.cacheMap[key]; ok {
		if !value.Expired() {
			return value.ip, nil
		}
	}

	return "", errors.New("cache missed")
}

func (d *DNSCache) Delete(key string) {
	d.cacheLock.Lock()
	delete(d.cacheMap, key)
	d.cacheLock.Unlock()
}
